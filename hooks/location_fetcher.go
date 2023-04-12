package hooks

import (
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/pkg/maze"
)

type LocationFetcher struct {
	language string
	fs       *fs.FS
	maze     *maze.Maze
}

func NewLocationFetcher(fs *fs.FS, language string) *LocationFetcher {
	return &LocationFetcher{
		language: language,
		fs:       fs,
		maze: maze.NewMaze(&http.Client{
			Timeout: time.Minute,
		}),
	}
}

func (l *LocationFetcher) EntryHook(_, e *eagle.Entry) error {
	if e.Listing != nil || e.Location != nil {
		return nil
	}

	return l.FetchLocation(e)
}

func (l *LocationFetcher) FetchLocation(e *eagle.Entry) error {
	if e.RawLocation == "" {
		return nil
	}

	location, err := l.parseLocation(e.RawLocation)
	if err != nil {
		return err
	}

	if location != nil {
		_, err = l.fs.TransformEntry(e.ID, func(e *eagle.Entry) (*eagle.Entry, error) {
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
	} else if strings.HasPrefix(str, "/") {
		_, err = l.fs.GetEntry(str)
		return nil, err
	} else {
		location, err = l.maze.Search(l.language, str)
	}

	if err != nil {
		return nil, err
	}

	return location, nil
}
