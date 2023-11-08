package linkding

import (
	"path/filepath"
	"reflect"

	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/pkg/linkding"
)

const (
	DefaultDataFileName = "bookmarks.json"
)

type BookmarksUpdater struct {
	dataFilename string
	client       *linkding.Linkding
	fs           *core.FS
}

func NewBookmarksUpdater(c *core.Linkding, fs *core.FS) *BookmarksUpdater {
	return &BookmarksUpdater{
		dataFilename: DefaultDataFileName,
		client:       linkding.NewLinkding(c.Endpoint, c.Key),
		fs:           fs,
	}
}

func (u *BookmarksUpdater) UpdateBookmarks() error {
	newBookmarks, err := u.client.Fetch()
	if err != nil {
		return err
	}

	filename := filepath.Join(core.DataDirectory, u.dataFilename)

	var oldBookmarks []linkding.Bookmark
	err = u.fs.ReadJSON(filename, &oldBookmarks)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(oldBookmarks, newBookmarks) {
		return nil
	}

	return u.fs.WriteJSON(filename, newBookmarks, "bookmarks: synchronize with linkding")
}
