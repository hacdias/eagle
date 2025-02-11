package locations

import (
	"net/http"
	"strings"
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

func NewLocations(co *core.Core, config map[string]interface{}) (server.Plugin, error) {
	cfg := typed.New(config)

	language := co.Language()
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

	locationStr := typed.New(e.Other).String("location")
	if locationStr == "" {
		return nil
	}

	location, err := l.parseLocation(locationStr)
	if err != nil {
		return err
	}

	if location == nil {
		return nil
	}

	e.Other["location"] = location
	return l.core.SaveEntry(e)
}

func (l *Locations) parseLocation(str string) (*maze.Location, error) {
	if strings.HasPrefix(str, "geo:") {
		return l.maze.ReverseGeoURI(l.language, str)
	} else if strings.HasPrefix(str, "airport:") {
		code := strings.TrimPrefix(str, "airport:")
		return l.maze.Airport(code)
	} else {
		return l.maze.Search(l.language, str)
	}

	// Also add swarm option
}
