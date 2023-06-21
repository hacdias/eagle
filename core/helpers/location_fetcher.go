package helpers

import (
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/core"
	"github.com/hacdias/maze"
)

type LocationFetcher struct {
	language string
	fs       *core.FS
	maze     *maze.Maze
}

func NewLocationFetcher(fs *core.FS, language string) *LocationFetcher {
	return &LocationFetcher{
		language: language,
		fs:       fs,
		maze: maze.NewMaze(&http.Client{
			Timeout: time.Minute,
		}),
	}
}

func (l *LocationFetcher) FetchLocation(e *core.Entry) error {
	if e.RawLocation == "" || e.Location != nil {
		return nil
	}

	location, err := l.parseLocation(e.RawLocation)
	if err != nil {
		return err
	}

	if location != nil {
		_, err = l.fs.TransformEntry(e.ID, "entry: add location info to "+e.ID, func(e *core.Entry) (*core.Entry, error) {
			e.RawLocation = ""
			e.Location = location
			return e, nil
		})
	}

	return err
}

func (l *LocationFetcher) parseLocation(str string) (*maze.Location, error) {
	var (
		location *maze.Location
		err      error
	)

	if strings.HasPrefix(str, "geo:") {
		location, err = l.maze.ReverseGeoURI(l.language, str)
	} else {
		location, err = l.maze.Search(l.language, str)
	}

	if err != nil {
		return nil, err
	}

	return location, nil
}
