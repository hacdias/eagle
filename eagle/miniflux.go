package eagle

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hacdias/eagle/v4/config"
	miniflux "miniflux.app/client"
)

type Feed struct {
	Title    string `json:"title"`
	Site     string `json:"site"`
	Feed     string `json:"feed"`
	Category string `json:"category"`
}

type Miniflux struct {
	*config.Miniflux
}

func (m *Miniflux) Fetch() ([]Feed, error) {
	client := miniflux.New(m.Endpoint, m.Key)

	rawFeeds, err := client.Feeds()
	if err != nil {
		return nil, err
	}

	sort.Slice(rawFeeds, func(i, j int) bool {
		return rawFeeds[i].Title < rawFeeds[j].Title
	})

	var feeds []Feed
	for _, feed := range rawFeeds {
		feeds = append(feeds, Feed{
			Title:    feed.Title,
			Feed:     feed.FeedURL,
			Site:     feed.SiteURL,
			Category: strings.ToLower(feed.Category.Title),
		})
	}

	return feeds, nil
}

const (
	blogrollEntryID  = "/links"
	blogrollFileName = ".feeds.json"
	blogrollTagStart = "<!--BLOGROLL-->"
	blogrollTagEnd   = "<!--/BLOGROLL-->"
)

func (e *Eagle) UpdateBlogroll() error {
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

func (e *Eagle) initBlogrollCron() error {
	if e.miniflux == nil {
		return nil
	}

	_, err := e.cron.AddFunc("CRON_TZ=UTC 00 00 * * *", func() {
		err := e.UpdateBlogroll()
		if err != nil {
			e.Notifier.Error(fmt.Errorf("blogroll updater: %w", err))
		}
	})

	return err
}

func minifluxFeedsToMarkdown(feeds []Feed) string {
	md := ""
	for _, feed := range feeds {
		if strings.ToLower(feed.Category) == "following" {
			md += fmt.Sprintf("- [%s](%s)\n", feed.Title, feed.Site)
		}
	}

	return md
}
