package blogroll

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/miniflux"
	"github.com/hacdias/eagle/v4/util"
)

// TODO: this is something that could likely be more configurable, or added as an "add-on" to eagle.
// For example, an interface that registers a function that is executed in a cron job. Same for summaries.

const (
	blogrollEntryID  = "/links"
	blogrollFileName = ".feeds.json"
	blogrollTagStart = "<!--BLOGROLL-->"
	blogrollTagEnd   = "<!--/BLOGROLL-->"
)

type MinifluxBlogrollUpdater struct {
	Client *miniflux.Miniflux
	Eagle  *eagle.Eagle // wip: remove this
}

func (u *MinifluxBlogrollUpdater) UpdateMinifluxBlogroll() error {
	feeds, err := u.Client.Fetch()
	if err != nil {
		return err
	}

	e, err := u.Eagle.TransformEntry(blogrollEntryID, func(e *entry.Entry) (*entry.Entry, error) {
		var err error
		md := minifluxFeedsToMarkdown(feeds)
		e.Content, err = util.ReplaceInBetween(e.Content, blogrollTagStart, blogrollTagEnd, md)
		return e, err
	})
	if err != nil {
		return err
	}

	filename := filepath.Join(eagle.ContentDirectory, e.ID, blogrollFileName)
	err = u.Eagle.FS.WriteJSON(filename, feeds, "update blogroll")
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
