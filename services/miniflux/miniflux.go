package miniflux

import (
	"path/filepath"
	"reflect"
	"time"

	"github.com/hacdias/eagle/core"
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
	fs           *core.FS
}

func NewBlogrollUpdater(c *core.Miniflux, fs *core.FS) *BlogrollUpdater {
	return &BlogrollUpdater{
		entryID:      DefaultEntryID,
		dataFilename: DefaultDataFileName,
		client:       miniflux.NewMiniflux(c.Endpoint, c.Key),
		fs:           fs,
	}
}

func (u *BlogrollUpdater) UpdateBlogroll() error {
	newFeeds, err := u.client.Fetch()
	if err != nil {
		return err
	}

	filename := filepath.Join(core.DataDirectory, u.dataFilename)

	var oldFeeds miniflux.Feeds
	err = u.fs.ReadJSON(filename, &oldFeeds)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(oldFeeds, newFeeds) {
		return nil
	}

	err = u.fs.WriteJSON(filename, newFeeds, "blogroll: synchronize with miniflux")
	if err != nil {
		return err
	}

	_, err = u.fs.TransformEntry(u.entryID, "blogroll: update entry modified date", func(e *core.Entry) (*core.Entry, error) {
		e.LastMod = time.Now()
		return e, err
	})

	return err
}
