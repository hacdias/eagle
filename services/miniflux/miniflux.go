package miniflux

import (
	"path/filepath"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/pkg/miniflux"
)

const (
	DefaultEntryID      = "/blogroll/"
	DefaultDataFileName = "feeds.json"
)

type BlogrollUpdater struct {
	entryID      string
	dataFilename string
	client       *miniflux.Miniflux
	fs           *fs.FS
}

func NewBlogrollUpdater(c *eagle.Miniflux, fs *fs.FS) *BlogrollUpdater {
	// TODO: make entryID and dataFilename configurable.
	return &BlogrollUpdater{
		entryID:      DefaultEntryID,
		dataFilename: DefaultDataFileName,
		client:       miniflux.NewMiniflux(c.Endpoint, c.Key),
		fs:           fs,
	}
}

func (u *BlogrollUpdater) UpdateBlogroll() error {
	feeds, err := u.client.Fetch()
	if err != nil {
		return err
	}

	filename := filepath.Join(fs.DataDirectory, u.entryID, u.dataFilename)
	err = u.fs.WriteJSON(filename, feeds)
	if err != nil {
		return err
	}

	_, err = u.fs.TransformEntry(u.entryID, func(e *eagle.Entry) (*eagle.Entry, error) {
		e.LastMod = time.Now()
		return e, err
	})

	return err
}
