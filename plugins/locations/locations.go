package locations

import (
	"net/http"
	"time"

	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
	"go.hacdias.com/maze"
)

var (
	_ server.HookPlugin = &Locations{}
)

func init() {
	server.RegisterPlugin("locations", NewLocations)
}

type Locations struct {
	core     *core.Core
	maze     *maze.Maze
	language string
	expand   bool
}

func NewLocations(co *core.Core, config map[string]any) (server.Plugin, error) {
	cfg := typed.New(config)

	language := co.SiteConfig().LanguageCode
	if language == "" {
		language = "en"
	}

	return &Locations{
		core: co,
		maze: maze.NewMaze(&http.Client{
			Timeout: time.Minute,
		}),
		language: language,
		expand:   cfg.Bool("expand"),
	}, nil
}

func (l *Locations) PreSaveHook(*core.Entry) error {
	return nil
}

func (l *Locations) PostSaveHook(e *core.Entry) error {
	if !l.expand {
		return nil
	}

	if e.Location == nil || e.Deleted() {
		return nil
	}

	hasDetails := e.Location.Country != "" || e.Location.Locality != "" || e.Location.Name != "" || e.Location.ICAO != "" || e.Location.IATA != ""
	if hasDetails {
		return nil
	}

	var err error
	e.Location, err = l.maze.ReverseGeoURI(l.language, e.Location.String())
	if err != nil {
		return err
	}

	return l.core.SaveEntry(e)
}
