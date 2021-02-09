package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
	"github.com/spf13/afero"
	"go.uber.org/zap"
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
}

func NewServer(c *config.Config, e *services.Eagle) *Server {
	server := &Server{
		SugaredLogger: c.S().Named("server"),
		Eagle:         e,
		eagle:         e,
		c:             c,
	}

	go func() {
		server.Info("waiting for new directories")
		for dir := range e.PublicDirCh {
			server.Infof("received new public directory: %s", dir)
			oldDir := server.dir

			// TODO: should this be locked somehow?
			server.dir = dir
			server.fs = afero.NewBasePathFs(afero.NewOsFs(), dir)
			server.httpdir = http.FileServer(neuteredFs{afero.NewHttpFs(server.fs).Dir("/")})

			err := os.RemoveAll(oldDir)
			if err != nil {
				server.Warnf("could not delete old directory: %s", err)
				server.NotifyError(err)
			}
		}
		server.Info("stopped waiting for new directories, channel closed")
	}()

	return server
}

func (s *Server) StartHTTP() error {
	r := chi.NewRouter()
	r.Use(s.recoverer)

	r.With(middleware.BasicAuth(s.c.Domain, s.c.BasicAuth)).Route("/dashboard", func(r chi.Router) {
		// Interface: r.Get("/")

		r.Get("/editor", s.editorGetHandler)
		r.Post("/editor", s.editorPostHandler)
	})

	r.Get("/editor", s.editorGetHandler)
	r.Delete("/editor", s.editorDeleteHandler)
	r.Post("/editor", s.editorPostHandler)

	r.Post("/webhook", s.webhookHandler)
	r.Post("/webmention", s.webmentionHandler)
	r.Post("/activitypub/inbox", s.activityPubPostInboxHandler)
	//r.Get("/search.json", s.searchHandler)

	// Make sure we have a built version!
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

	static := s.staticHandler()

	r.NotFound(static)
	r.MethodNotAllowed(static)

	s.Infof("Listening on http://localhost:%d", s.c.Port)
	return http.ListenAndServe(":"+strconv.Itoa(s.c.Port), r)
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
