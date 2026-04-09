package locations

import (
	"net/http"
	"time"

	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/server"
	"go.hacdias.com/maze"
	"go.uber.org/zap"
)

var (
	_ server.HookPlugin = &Locations{}
)

func init() {
	server.RegisterPlugin("locations", NewLocations)
}

type Locations struct {
	log      *zap.SugaredLogger
	maze     *maze.Maze
	language string
	expand   bool
}

func NewLocations(co *core.Core, config map[string]any) (server.Plugin, error) {
	cfg := typed.New(config)

	language := co.SiteConfig().Locale
	if language == "" {
		language = "en"
	}

	return &Locations{
		maze: maze.NewMaze(&http.Client{
			Timeout: time.Minute,
		}),
		language: language,
		expand:   cfg.Bool("expand"),
		log:      log.S().Named("locations"),
	}, nil
}

func (l *Locations) PreSaveHook(e *core.Entry) error {
	if !l.expand {
		return nil
	}

	if e.Location == nil || e.Deleted() {
		return nil
	}

	hasDetails := e.Location.Country != "" ||
		e.Location.CountryCode != "" ||
		e.Location.Locality != "" ||
		e.Location.ICAO != "" ||
		e.Location.IATA != "" ||
		e.Location.PostalCode != "" ||
		e.Location.Region != ""
	if hasDetails {
		return nil
	}

	if loc, err := l.maze.ReverseGeoURI(l.language, e.Location.String()); err == nil {
		e.Location = loc
	} else {
		l.log.Warnf("failed to fetch location", "entry", e.ID, "err", err)
	}

	return nil
}

func (l *Locations) PostSaveHook(e *core.Entry, _ bool) error {
	return nil
}
