package server

import (
	"html/template"
	"net/http"
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
