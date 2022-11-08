package eagle

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v4/pkg/miniflux"
)

// TODO: this is something that could likely be more configurable, or added as an "add-on" to eagle.
// For example, an interface that registers a function that is executed in a cron job. Same for summaries.

const (
	blogrollEntryID  = "/links"
	blogrollFileName = ".feeds.json"
	blogrollTagStart = "<!--BLOGROLL-->"
	blogrollTagEnd   = "<!--/BLOGROLL-->"
)

func (e *Eagle) UpdateMinifluxBlogroll() error {
	if e.miniflux == nil {
		return errors.New("miniflux is not implemented")
	}

	feeds, err := e.miniflux.Fetch()
	if err != nil {
		return err
	}

	ee, err := e.GetEntry(blogrollEntryID)
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, ee.ID, blogrollFileName)
	err = e.fs.WriteJSON(filename, feeds, "update blogroll")
	if err != nil {
		return err
	}

	md := minifluxFeedsToMarkdown(feeds)
	ee.Content, err = replaceBetween(ee.Content, blogrollTagStart, blogrollTagEnd, md)
	if err != nil {
		return err
	}

	err = e.saveEntry(ee)
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}

func (e *Eagle) initMinifluxCron() error {
	if e.miniflux == nil {
		return nil
	}

	_, err := e.cron.AddFunc("CRON_TZ=UTC 00 00 * * *", func() {
		err := e.UpdateMinifluxBlogroll()
		if err != nil {
			e.Notifier.Error(fmt.Errorf("blogroll updater: %w", err))
		}
	})

	return err
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
