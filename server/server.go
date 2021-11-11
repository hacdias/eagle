package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/log"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

type contextKey string

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
	jwtAuth      *jwtauth.JWTAuth
}

func NewServer(e *eagle.Eagle) (*Server, error) {
	s := &Server{
		Eagle:   e,
		log:     log.S().Named("server"),
		servers: []*httpServer{},
	}

	secret := base64.StdEncoding.EncodeToString([]byte(e.Config.Auth.Secret))
	s.jwtAuth = jwtauth.New("HS256", []byte(secret), nil)

	return s, nil
}

func (s *Server) Start() error {
	errCh := make(chan error)
	router := s.makeRouter()

	// Start server(s)
	err := s.startRegularServer(errCh, router)
	if err != nil {
		return err
	}

	if s.Config.Tor != nil {
		err = s.startTor(errCh, router)
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

func (s *Server) startRegularServer(errCh chan error, h http.Handler) error {
	addr := ":" + strconv.Itoa(s.Config.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := &http.Server{Handler: h}
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
				s.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// borrowed from chi + redirection.
func cleanPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())

		routePath := rctx.RoutePath
		if routePath == "" {
			if r.URL.RawPath != "" {
				routePath = r.URL.RawPath
			} else {
				routePath = r.URL.Path
			}
			routePath = path.Clean(routePath)
		}

		if r.URL.Path != routePath {
			http.Redirect(w, r, routePath, http.StatusTemporaryRedirect)
			return
		}

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
		// TODO: maybe notify as these are terminal errors kinda.
		s.log.Error("error while serving json", err)
	}
}

func (s *Server) serveErrorJSON(w http.ResponseWriter, code int, err, errDescription string) {
	s.serveJSON(w, code, map[string]interface{}{
		"error":             err,
		"error_description": errDescription,
	})
}

func (s *Server) serveHTMLWithStatus(w http.ResponseWriter, r *http.Request, data *eagle.RenderData, tpls []string, code int) {
	data.TorUsed = s.isUsingTor(r)
	data.OnionAddress = s.onionAddress
	data.LoggedIn = s.isLoggedIn(w, r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)

	err := s.Render(w, data, tpls)
	if err != nil {
		// TODO: maybe notify as these are terminal errors kinda.
		s.log.Error("error while serving HTML", err)
	}
}

func (s *Server) serveHTML(w http.ResponseWriter, r *http.Request, data *eagle.RenderData, tpls []string) {
	s.serveHTMLWithStatus(w, r, data, tpls, http.StatusOK)
}

func (s *Server) serveErrorHTML(w http.ResponseWriter, r *http.Request, code int, err error) {
	if err != nil {
		s.log.Error(err)
	}

	w.Header().Del("Cache-Control")

	data := &eagle.RenderData{
		Entry: &entry.Entry{
			Frontmatter: entry.Frontmatter{
				Title: fmt.Sprintf("%d %s", code, http.StatusText(code)),
			},
			Content: strconv.Itoa(code),
		},
	}

	s.serveHTMLWithStatus(w, r, data, []string{eagle.TemplateError}, code)
}
