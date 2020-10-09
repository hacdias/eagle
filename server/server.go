package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
	"go.uber.org/zap"
)

type Server struct {
	sync.Mutex
	*zap.SugaredLogger
	*services.Services
	c *config.Config
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
