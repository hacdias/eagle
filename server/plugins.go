package server

import (
	"fmt"
	"net/http"

	"go.hacdias.com/eagle/core"
)

type PluginInitializer func(fs *core.FS, config map[string]interface{}) (Plugin, error)

type Plugin interface {
	GetWebHandler() (string, http.HandlerFunc)
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
