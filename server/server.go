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
	"github.com/hacdias/eagle/services"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Server struct {
	sync.Mutex
	*zap.SugaredLogger
	*services.Eagle
	c *config.Config

	eagle *services.Eagle

	dir     string
	fs      afero.Fs
	httpdir http.Handler
	server  *http.Server

	bot *tb.Bot
}

func NewServer(c *config.Config, e *services.Eagle) (*Server, error) {
	s := &Server{
		SugaredLogger: c.S().Named("server"),
		Eagle:         e,
		eagle:         e,
		c:             c,
	}

	err := s.buildBot()
	if err != nil {
		return nil, err
	}

	basicauth := middleware.BasicAuth(c.Domain, c.BasicAuth)

	r := chi.NewRouter()
	r.Use(s.recoverer)

	r.With(basicauth).Route("/sorcery", func(r chi.Router) {
		// Interface: r.Get("/")

		r.Get("/editor", s.editorGetHandler)
		r.Post("/editor", s.editorPostHandler)
	})

	//r.Get("/search.json", s.searchHandler)
	r.Post("/webhook", s.webhookHandler)
	r.Post("/webmention", s.webmentionHandler)
	r.Post("/activitypub/inbox", s.activityPubPostInboxHandler)

	static := s.staticHandler()

	r.NotFound(static)
	r.MethodNotAllowed(static)

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
		oldDir := s.dir

		s.dir = dir
		s.fs = afero.NewBasePathFs(afero.NewOsFs(), dir)
		s.httpdir = http.FileServer(neuteredFs{afero.NewHttpFs(s.fs).Dir("/")})

		err := os.RemoveAll(oldDir)
		if err != nil {
			s.Warnf("could not delete old directory: %s", err)
			s.NotifyError(err)
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
