package server

import (
	"fmt"
	"net/http"

	"go.hacdias.com/eagle/core"
)

type PluginInitializer func(fs *core.FS, config map[string]interface{}) (Plugin, error)

type PluginWebUtilities struct {
	s *Server
}

func (u *PluginWebUtilities) JSON(w http.ResponseWriter, code int, data interface{}) {
	u.s.serveJSON(w, code, data)
}

func (u *PluginWebUtilities) ErrorJSON(w http.ResponseWriter, code int, err, errDescription string) {
	u.s.serveErrorJSON(w, code, err, errDescription)
}

func (u *PluginWebUtilities) ErrorHTML(w http.ResponseWriter, r *http.Request, code int, reqErr error) {
	u.s.serveErrorHTML(w, r, code, reqErr)
}

type Plugin interface {
	GetWebHandler(utils *PluginWebUtilities) (string, http.HandlerFunc)
	GetAction() (string, func() error)
	GetDailyCron() func() error
}

var (
	pluginRegistry = map[string]PluginInitializer{}
)

func RegisterPlugin(name string, pluginInitializer PluginInitializer) {
	if _, ok := pluginRegistry[name]; ok {
		panic(fmt.Sprintf("plugin with name  '%q' is already registered", name))
	}

	pluginRegistry[name] = pluginInitializer
}
