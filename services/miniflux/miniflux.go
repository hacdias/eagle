package miniflux

import (
	"path/filepath"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/pkg/miniflux"
)

const (
	DefaultEntryID   = "/blogroll"
	blogrollFileName = ".feeds.json"
)

type BlogrollUpdater struct {
	entryID string
	client  *miniflux.Miniflux
	fs      *fs.FS
}

func NewBlogrollUpdater(c *eagle.Miniflux, fs *fs.FS) *BlogrollUpdater {
	// TODO: make entry ID configurable.

	return &BlogrollUpdater{
		entryID: DefaultEntryID,
		client:  miniflux.NewMiniflux(c.Endpoint, c.Key),
		fs:      fs,
	}
}

func (u *BlogrollUpdater) UpdateBlogroll() error {
	feeds, err := u.client.Fetch()
	if err != nil {
		return err
	}

	filename := filepath.Join(fs.ContentDirectory, u.entryID, blogrollFileName)
	err = u.fs.WriteJSON(filename, feeds)
	if err != nil {
		return err
	}

	_, err = u.fs.TransformEntry(u.entryID, func(e *eagle.Entry) (*eagle.Entry, error) {
		e.Updated = time.Now()
		return e, err
	})
	return err
}
