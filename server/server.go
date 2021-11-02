package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/jwtauth"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/logging"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

type httpServer struct {
	Name string
	*http.Server
}

type Server struct {
	*eagle.Eagle
	log *zap.SugaredLogger

	serversLock sync.Mutex
	servers     []*httpServer

	onionAddress string

	// Dashboard-specific variables.
	token     *jwtauth.JWTAuth
	templates map[string]*template.Template
}

func NewServer(e *eagle.Eagle) (*Server, error) {
	s := &Server{
		Eagle:   e,
		log:     logging.S().Named("server"),
		servers: []*httpServer{},
	}

	if e.Config.Auth != nil {
		secret := base64.StdEncoding.EncodeToString([]byte(e.Config.Auth.Secret))
		s.token = jwtauth.New("HS256", []byte(secret), nil)
	}

	return s, nil
}

func (s *Server) Start() error {
	errCh := make(chan error)

	// Start server(s)
	err := s.startRegularServer(errCh)
	if err != nil {
		return err
	}

	if s.Config.Tailscale != nil {
		err = s.startTailscaleServer(errCh)
		if err != nil {
			return err
		}
	}

	if s.Config.Tor != nil {
		err = s.startTor(errCh)
		if err != nil {
			err = fmt.Errorf("onion service failed to start: %w", err)
			s.log.Error(err)
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
	for _, srv := range s.servers {
		s.log.Infof("shutting down %s", srv.Name)
		errs = multierror.Append(errs, srv.Shutdown(ctx))
	}
	return errs.ErrorOrNil()
}

func (s *Server) registerServer(srv *http.Server, name string) {
	s.serversLock.Lock()
	defer s.serversLock.Unlock()

	s.servers = append(s.servers, &httpServer{
		Server: srv,
		Name:   name,
	})
}

func (s *Server) startRegularServer(errCh chan error) error {
	addr := ":" + strconv.Itoa(s.Config.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	noDashboard := false
	if s.Config.Tailscale != nil {
		noDashboard = s.Config.Tailscale.ExclusiveDashboard
	}

	router := s.makeRouter(noDashboard)
	srv := &http.Server{Handler: router}

	s.registerServer(srv, "public")

	go func() {
		s.log.Infof("listening on %s", ln.Addr().String())
		errCh <- srv.Serve(ln)
	}()

	return nil
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				err := fmt.Errorf("panic while serving: %v: %s", rvr, string(debug.Stack()))
				s.NotifyError(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.log.Error("error while serving json", err)
	}
}

func (s *Server) serveErrorJSON(w http.ResponseWriter, code int, err error) {
	s.serveJSON(w, code, map[string]interface{}{
		"error":             http.StatusText(code),
		"error_description": err.Error(),
	})
}

func (s *Server) serveError(w http.ResponseWriter, code int, err error) {
	// TODO: render error template.

	if err != nil {
		// Do something
		s.log.Error(err)
	}

	bytes := []byte(http.StatusText(code))

	w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Del("Cache-Control")

	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(bytes)
}
