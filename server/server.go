package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/jwtauth"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/logging"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

type Server struct {
	*zap.SugaredLogger

	c *config.Config
	e *eagle.Eagle

	serversLock sync.Mutex
	servers     map[string]*http.Server

	// Dashboard-specific variables.
	token     *jwtauth.JWTAuth
	templates map[string]*template.Template

	// Website-specific variables.
	staticFsLock sync.RWMutex
	staticFs     *staticFs
}

func NewServer(c *config.Config, e *eagle.Eagle) (*Server, error) {
	s := &Server{
		SugaredLogger: logging.S().Named("server"),
		servers:       map[string]*http.Server{},
		e:             e,
		c:             c,
	}

	if c.Auth != nil {
		secret := base64.StdEncoding.EncodeToString([]byte(c.Auth.Secret))
		s.token = jwtauth.New("HS256", []byte(secret), nil)
	}

	return s, nil
}

func (s *Server) Start() error {
	// Start public dir worker
	go s.publicDirWorker()

	// Make sure we have a built version to serve
	should, err := s.e.ShouldBuild()
	if err != nil {
		return err
	}

	if should {
		err = s.e.Build(false)
		if err != nil {
			return err
		}
	}

	errCh := make(chan error)

	// Start server(s)
	err = s.startRegularServer(errCh)
	if err != nil {
		return err
	}

	if s.c.Tailscale != nil {
		err = s.startTailscaleServer(errCh)
		if err != nil {
			return err
		}
	}

	// Collect errors when the server stops
	var errs *multierror.Error
	for i := 0; i < len(s.servers); i++ {
		errs = multierror.Append(errs, <-errCh)
	}
	return errs.ErrorOrNil()
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var errs *multierror.Error
	for name, srv := range s.servers {
		s.Infof("shutting down %s", name)
		errs = multierror.Append(errs, srv.Shutdown(ctx))
	}
	return errs.ErrorOrNil()
}

func (s *Server) registerServer(srv *http.Server, name string) error {
	s.serversLock.Lock()
	defer s.serversLock.Unlock()

	if _, ok := s.servers[name]; ok {
		return fmt.Errorf("server %s already registered", name)
	}

	s.servers[name] = srv
	return nil
}

func (s *Server) startRegularServer(errCh chan error) error {
	addr := ":" + strconv.Itoa(s.c.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	noDashboard := false
	if s.c.Tailscale != nil {
		noDashboard = s.c.Tailscale.ExclusiveDashboard
	}

	router := s.makeRouter(noDashboard)
	srv := &http.Server{Handler: router}

	err = s.registerServer(srv, "public")
	if err != nil {
		return err
	}

	go func() {
		s.Infof("listening on %s", ln.Addr().String())
		errCh <- srv.Serve(ln)
	}()

	return nil
}

func (s *Server) publicDirWorker() {
	s.Info("waiting for new directories")
	for dir := range s.e.PublicDirCh {
		s.Infof("received new public directory: %s", dir)

		s.staticFsLock.Lock()
		oldFs := s.staticFs
		s.staticFs = newStaticFs(dir)
		s.staticFsLock.Unlock()

		if oldFs != nil {
			err := os.RemoveAll(oldFs.dir)
			if err != nil {
				s.Warn("could not delete old directory", err)
				s.e.NotifyError(err)
			}
		}
	}
	s.Info("stopped waiting for new directories, channel closed")
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				s.Errorf("panic while serving: %v: %s", rvr, string(debug.Stack()))
				s.e.NotifyError(fmt.Errorf(fmt.Sprint(rvr)))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (s *Server) headers(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (s *Server) serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.Error("error while serving json", err)
	}
}

func (s *Server) serveError(w http.ResponseWriter, code int, err error) {
	s.serveJSON(w, code, map[string]interface{}{
		"error":             http.StatusText(code),
		"error_description": err.Error(),
	})
}
