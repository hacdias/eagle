package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.hacdias.com/eagle/core"
)

type PluginInitializer func(co *core.Core, config map[string]any) (Plugin, error)

type PluginWebUtilities struct {
	s *Server
}

func (u *PluginWebUtilities) JSON(w http.ResponseWriter, code int, data any) {
	u.s.serveJSON(w, code, data)
}

func (u *PluginWebUtilities) ErrorHTML(w http.ResponseWriter, r *http.Request, code int, reqErr error) {
	u.s.serveErrorHTML(w, r, code, reqErr)
}

func (u *PluginWebUtilities) ReadStaticFile(path string) ([]byte, error) {
	return u.s.staticFs.ReadFile(path)
}

type Plugin = any

// TODO: support multiple actions per plugin.
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

type PreSaveHookPlugin interface {
	PreSaveHook(*core.Entry) error
}

type PostSaveHookPlugin interface {
	PostSaveHook(e *core.Entry, isNew bool) error
}

type Photo struct {
	Data     []byte
	Title    string
	Alt      string
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

type QueuePlugin interface {
	QueueItemType() string
	HandleQueueItem(ctx context.Context, payload []byte) error
}

type MCPPlugin interface {
	RegisterMCPTools(*mcp.Server)
}

var (
	pluginRegistry = map[string]PluginInitializer{}
)

// RegisterPlugin registers a plugin whose configuration is decoded into C.
func RegisterPlugin[C any](name string, init func(co *core.Core, config C) (Plugin, error)) {
	if _, ok := pluginRegistry[name]; ok {
		panic(fmt.Sprintf("plugin '%q' is already registered", name))
	}

	pluginRegistry[name] = func(co *core.Core, configMap map[string]any) (Plugin, error) {
		var config C
		err := mapstructure.Decode(configMap, &config)
		if err != nil {
			return nil, fmt.Errorf("plugin %s: %w", name, err)
		}

		return init(co, config)
	}
}
