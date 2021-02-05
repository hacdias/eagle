package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
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

	tpl *template.Template

	dir     string
	fs      afero.Fs
	httpdir http.Handler
}

func NewServer(c *config.Config, s *services.Services) (*Server, error) {
	tpl, err := template.ParseGlob(filepath.Join(c.Source, "templates", "*.tmpl"))
	if err != nil {
		return nil, err
	}

	server := &Server{
		SugaredLogger: c.S().Named("server"),
		Services:      s,
		c:             c,
		tpl:           tpl,
	}

	return server, nil
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
