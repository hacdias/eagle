package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/logging"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Server struct {
	//sync.Mutex
	*eagle.Eagle
	*zap.SugaredLogger

	c      *config.Config
	bot    *tb.Bot
	server *http.Server

	staticFsLock sync.RWMutex
	staticFs     *staticFs
}

func NewServer(c *config.Config, e *eagle.Eagle) (*Server, error) {
	s := &Server{
		SugaredLogger: logging.S().Named("server"),
		Eagle:         e,
		c:             c,
	}

	err := s.buildBot()
	if err != nil {
		return nil, err
	}

	basicauth := middleware.BasicAuth(c.Domain, c.BasicAuth)

	r := chi.NewRouter()
	r.Use(s.recoverer)
	r.Use(s.headers)

	r.With(basicauth).Route(dashboardPath, func(r chi.Router) {
		fs := afero.NewBasePathFs(afero.NewOsFs(), "dashboard/static")
		httpdir := http.FileServer(neuteredFs{afero.NewHttpFs(fs).Dir("/")})

		r.Get("/", s.dashboardHandler)
		r.Get("/editor", s.editorGetHandler)
		r.Post("/editor", s.editorPostHandler)
		r.Get("/*", http.StripPrefix(dashboardPath, httpdir).ServeHTTP)
	})

	r.Get("/search.json", s.searchHandler)
	r.Post("/webhook", s.webhookHandler)
	r.Post("/webmention", s.webmentionHandler)
	r.Post("/activitypub/inbox", s.activityPubPostInboxHandler)

	r.Get("/*", s.staticHandler)
	r.NotFound(s.staticHandler)         // NOTE: maybe repetitive regarding previous line.
	r.MethodNotAllowed(s.staticHandler) // NOTE: maybe useless.

	s.server = &http.Server{
		Addr:    ":" + strconv.Itoa(s.c.Port),
		Handler: r,
	}

	return s, nil
}

func (s *Server) Start() error {
	// Start bot and public dir worker
	go s.bot.Start()
	go s.publicDirWorker()

	// Make sure we have a built version to serve
	should, err := s.ShouldBuild()
	if err != nil {
		return err
	}

	if should {
		err = s.Build(false)
		if err != nil {
			return err
		}
	}

	s.Infof("Listening on http://localhost:%d", s.c.Port)
	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.bot.Stop()
	return s.server.Shutdown(ctx)
}

func (s *Server) publicDirWorker() {
	s.Info("waiting for new directories")
	for dir := range s.PublicDirCh {
		s.Infof("received new public directory: %s", dir)

		s.staticFsLock.Lock()
		oldFs := s.staticFs
		s.staticFs = newStaticFs(dir)
		s.staticFsLock.Unlock()

		if oldFs != nil {
			err := os.RemoveAll(oldFs.dir)
			if err != nil {
				s.Warnf("could not delete old directory: %s", err)
				s.NotifyError(err)
			}
		}
	}
	s.Info("stopped waiting for new directories, channel closed")
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				s.Errorf("panic while serving: %s", rvr)
				s.NotifyError(fmt.Errorf(fmt.Sprint(rvr)))
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
		s.Errorf("error while serving json: %s", err)
	}
}

func (s *Server) serveError(w http.ResponseWriter, code int, err error) {
	s.serveJSON(w, code, map[string]interface{}{
		"error":             http.StatusText(code),
		"error_description": err.Error(),
	})
}
