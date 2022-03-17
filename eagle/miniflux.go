package eagle

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hacdias/eagle/v3/config"
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

	feeds := []Feed{}

	for _, feed := range rawFeeds {
		feeds = append(feeds, Feed{
			Title:    feed.Title,
			Feed:     feed.FeedURL,
			Site:     feed.SiteURL,
			Category: strings.ToLower(feed.Category.Title),
		})
	}

	sort.Slice(feeds, func(i, j int) bool {
		return feeds[i].Title < feeds[j].Title
	})

	return feeds, nil
}

func (e *Eagle) UpdateBlogroll() error {
	if e.miniflux == nil {
		return errors.New("miniflux is not implemented")
	}

	feeds, err := e.miniflux.Fetch()
	if err != nil {
		return err
	}

	// TODO: do not like this hardcoded.
	filename := filepath.Join(ContentDirectory, "blogroll/_blogroll.json")

	err = e.fs.WriteJSON(filename, feeds, "update blogroll")
	if err != nil {
		return err
	}

	ee, err := e.GetEntry("/blogroll")
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}
