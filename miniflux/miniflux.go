package miniflux

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/pkg/miniflux"
	"github.com/hacdias/eagle/v4/util"
)

const (
	DefaultEntryID = "/links"

	blogrollFileName = ".feeds.json"
	blogrollTagStart = "<!--BLOGROLL-->"
	blogrollTagEnd   = "<!--/BLOGROLL-->"
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

	e, err := u.fs.TransformEntry(u.entryID, func(e *eagle.Entry) (*eagle.Entry, error) {
		var err error
		md := minifluxFeedsToMarkdown(feeds)
		e.Content, err = util.ReplaceInBetween(e.Content, blogrollTagStart, blogrollTagEnd, md)
		return e, err
	})
	if err != nil {
		return err
	}

	filename := filepath.Join(fs.ContentDirectory, e.ID, blogrollFileName)
	err = u.fs.WriteJSON(filename, feeds, "update blogroll")
	if err != nil {
		return err
	}

	return nil
}

func minifluxFeedsToMarkdown(feeds []miniflux.Feed) string {
	md := ""
	for _, feed := range feeds {
		if strings.ToLower(feed.Category) == "following" {
			md += fmt.Sprintf("- [%s](%s)\n", feed.Title, feed.Site)
		}
	}

	return md
}
