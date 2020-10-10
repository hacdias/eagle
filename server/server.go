package server

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
	"go.uber.org/zap"
)

type Server struct {
	sync.Mutex
	*zap.SugaredLogger
	*services.Services
	c       *config.Config
	httpdir http.Dir
}

func NewServer(log *zap.SugaredLogger, c *config.Config, s *services.Services) *Server {
	server := &Server{
		SugaredLogger: log,
		Services:      s,
		c:             c,
	}

	go func() {
		log.Info("waiting for new directories")
		for dir := range s.PublicDirChanges {
			log.Infof("received new public directory: %s", dir)
			oldDir := string(server.httpdir)
			server.httpdir = http.Dir(dir)

			err := os.RemoveAll(oldDir)
			if err != nil {
				log.Warnf("could not delete old directory: %s", err)
				s.Notify.Error(err)
			}
		}
		log.Info("stopped waiting for new directories, channel closed")
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
