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

	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/dashboard/static"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/logging"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

type Server struct {
	*zap.SugaredLogger

	c *config.Config
	e *eagle.Eagle

	serversMu sync.Mutex
	servers   []*http.Server

	// Dashboard-specific variables.
	token     *jwtauth.JWTAuth
	templates map[string]*template.Template

	// Website-specific variables.
	staticFsLock sync.RWMutex
	staticFs     *staticFs
}

func NewServer(c *config.Config, e *eagle.Eagle) (*Server, error) {
	secret := base64.StdEncoding.EncodeToString([]byte(c.Auth.Secret))
	token := jwtauth.New("HS256", []byte(secret), nil)

	s := &Server{
		SugaredLogger: logging.S().Named("server"),
		e:             e,
		c:             c,
		token:         token,
	}

	return s, nil
}

func (s *Server) makeWebsiteServer() http.Handler {
	r := chi.NewRouter()
	r.Use(s.recoverer)
	r.Use(s.headers)

	r.Get("/search.json", s.searchHandler)
	r.Post("/webhook", s.webhookHandler)
	r.Post("/webmention", s.webmentionHandler)

	r.Get("/*", s.staticHandler)
	r.NotFound(s.staticHandler)         // NOTE: maybe repetitive regarding previous line.
	r.MethodNotAllowed(s.staticHandler) // NOTE: maybe useless.

	return r
}

func (s *Server) makeDashboardHandler() http.Handler {
	r := chi.NewRouter()
	r.Use(s.recoverer)
	r.Use(s.headers)

	fs := http.FS(static.FS)
	if s.c.Development {
		fs = http.FS(afero.NewIOFS(afero.NewBasePathFs(afero.NewOsFs(), "./dashboard/static")))
	}

	httpdir := http.FileServer(neuteredFs{fs})

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(s.token))
		r.Use(s.dashboardAuth)

		r.Get("/", s.dashboardGetHandler)
		r.Get("/new", s.newGetHandler)
		r.Get("/edit", s.editGetHandler)
		r.Get("/reply", s.replyGetHandler)
		r.Get("/delete", s.deleteGetHandler)
		r.Get("/reshare", s.reshareGetHandler)
		r.Get("/blogroll", s.blogrollGetHandler)
		r.Get("/gedit", s.geditGetHandler)

		r.Post("/", s.dashboardPostHandler)
		r.Post("/new", s.newPostHandler)
		r.Post("/edit", s.editPostHandler)
		r.Post("/delete", s.deletePostHandler)
		r.Post("/reshare", s.resharePostHandler)
		r.Post("/gedit", s.geditPostHandler)
	})

	r.Get("/logout", s.logoutGetHandler)
	r.Get("/login", s.loginGetHandler)
	r.Post("/login", s.loginPostHandler)
	r.Get("/*", httpdir.ServeHTTP)

	return r
}

func (s *Server) startServer(h http.Handler, ln net.Listener, errCh chan error) {
	srv := &http.Server{Handler: h}

	s.serversMu.Lock()
	s.servers = append(s.servers, srv)
	s.serversMu.Unlock()

	s.Infof("Listening on %s", ln.Addr().String())
	errCh <- srv.Serve(ln)
}

func (s *Server) getTcpListener(port int) (net.Listener, error) {
	addr := ":" + strconv.Itoa(port)
	return net.Listen("tcp", addr)
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

	// Start Website Server
	wr := s.makeWebsiteServer()
	ln, err := s.getTcpListener(s.c.WebsitePort)
	if err != nil {
		return err
	}
	go s.startServer(wr, ln, errCh)

	// Start Dashboard Server
	dr := s.makeDashboardHandler()
	ln, err = s.getTcpListener(s.c.DashboardPort)
	if err != nil {
		return err
	}
	go s.startServer(dr, ln, errCh)

	// Collect errors in the end
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
	for _, s := range s.servers {
		errs = multierror.Append(errs, s.Shutdown(ctx))
	}
	return errs.ErrorOrNil()
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
				s.Warnf("could not delete old directory: %w", err)
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
		s.Errorf("error while serving json: %w", err)
	}
}

func (s *Server) serveError(w http.ResponseWriter, code int, err error) {
	s.serveJSON(w, code, map[string]interface{}{
		"error":             http.StatusText(code),
		"error_description": err.Error(),
	})
}
