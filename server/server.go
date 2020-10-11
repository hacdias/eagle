package server

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

type Server struct {
	sync.Mutex
	*zap.SugaredLogger
	*services.Services
	c *config.Config

	dir     string
	fs      afero.Fs
	httpdir http.Handler
}

func NewServer(c *config.Config, s *services.Services) *Server {
	server := &Server{
		SugaredLogger: c.S().Named("server"),
		Services:      s,
		c:             c,
	}

	go func() {
		server.Info("waiting for new directories")
		for dir := range s.PublicDirChanges {
			server.Infof("received new public directory: %s", dir)
			oldDir := server.dir

			// TODO: should this be locked somehow?
			server.dir = dir
			server.fs = afero.NewBasePathFs(afero.NewOsFs(), dir)
			server.httpdir = http.FileServer(neuteredFs{afero.NewHttpFs(server.fs).Dir("/")})

			err := os.RemoveAll(oldDir)
			if err != nil {
				server.Warnf("could not delete old directory: %s", err)
				s.Notify.Error(err)
			}
		}
		server.Info("stopped waiting for new directories, channel closed")
	}()

	return server
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
