package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"go.hacdias.com/eagle/config"
	"go.hacdias.com/eagle/fs"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/render"

	"go.uber.org/zap"
)

type contextKey string

type Server struct {
	cfg      *config.Config
	log      *zap.SugaredLogger
	server   *http.Server
	fs       *fs.FS
	renderer *render.Renderer
}

func NewServer(cfg *config.Config) (*Server, error) {
	renderer, err := render.NewRenderer(cfg)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:      cfg,
		log:      log.S().Named("server"),
		fs:       fs.NewFS(cfg.Server.Source, cfg.Site.BaseURL),
		renderer: renderer,
	}

	return s, nil
}

func (s *Server) Start() error {
	// Start server
	addr := ":" + strconv.Itoa(s.cfg.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	errCh := make(chan error)
	s.server = &http.Server{Handler: s.makeRouter()}
	go func() {
		s.log.Infof("listening on %s", ln.Addr().String())
		errCh <- s.server.Serve(ln)
	}()

	return <-errCh
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

func (s *Server) withRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				err := fmt.Errorf("panic while serving: %v: %s", rvr, string(debug.Stack()))
				s.log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func withCleanPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := path.Clean(r.URL.Path)
		if path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			path += "/"
		}

		if r.URL.Path != path {
			http.Redirect(w, r, path, http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}
