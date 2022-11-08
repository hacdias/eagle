package miniflux

import (
	"sort"
	"strings"

	miniflux "miniflux.app/client"
)

type Feed struct {
	Title    string `json:"title"`
	Site     string `json:"site"`
	Feed     string `json:"feed"`
	Category string `json:"category"`
}

type Miniflux struct {
	endpoint string
	key      string
}

func NewMiniflux(endpoint, key string) *Miniflux {
	return &Miniflux{
		endpoint: endpoint,
		key:      key,
	}
}

func (m *Miniflux) Fetch() ([]Feed, error) {
	client := miniflux.New(m.endpoint, m.key)

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
