package server

import (
	"context"
	"fmt"
	"net/http"

	"go.hacdias.com/eagle/core"
)

type PluginInitializer func(co *core.Core, config map[string]any) (Plugin, error)

type PluginWebUtilities struct {
	s *Server
}

func (u *PluginWebUtilities) JSON(w http.ResponseWriter, code int, data any) {
	u.s.serveJSON(w, code, data)
}

func (u *PluginWebUtilities) ErrorJSON(w http.ResponseWriter, code int, err, errDescription string) {
	u.s.serveErrorJSON(w, code, err, errDescription)
}

func (u *PluginWebUtilities) ErrorHTML(w http.ResponseWriter, r *http.Request, code int, reqErr error) {
	u.s.serveErrorHTML(w, r, code, reqErr)
}

type Plugin = any

type ActionPlugin interface {
	ActionName() string
	Action() error
}

type HandlerPlugin interface {
	HandlerRoute() string
	Handler(http.ResponseWriter, *http.Request, *PluginWebUtilities)
}

type CronPlugin interface {
	DailyCron() error
}

type HookPlugin interface {
	PreSaveHook(*core.Entry) error
	PostSaveHook(*core.Entry) error
}

type Photo struct {
	Data     []byte
	Title    string
	MimeType string
	Width    int
	Height   int
}

type SyndicationContext struct {
	Thumbnail *Photo
	Status    string
	Photos    []*Photo
}

type Syndicator struct {
	UID     string
	Name    string
	Default bool
}

type SyndicationPlugin interface {
	Syndicator() Syndicator
	IsSyndicated(*core.Entry) bool
	Syndicate(context.Context, *core.Entry, *SyndicationContext) error
}

var (
	pluginRegistry = map[string]PluginInitializer{}
)

func RegisterPlugin(name string, pluginInitializer PluginInitializer) {
	if _, ok := pluginRegistry[name]; ok {
		panic(fmt.Sprintf("plugin '%q' is already registered", name))
	}

	pluginRegistry[name] = pluginInitializer
}
