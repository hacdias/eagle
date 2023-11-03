package miniflux

import (
	"sort"
	"strings"

	miniflux "miniflux.app/client"
)

type Feeds map[string][]Feed

type Feed struct {
	Title string `json:"title"`
	Site  string `json:"site"`
	Feed  string `json:"feed"`
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

func (m *Miniflux) Fetch() (Feeds, error) {
	client := miniflux.New(m.endpoint, m.key)

	rawFeeds, err := client.Feeds()
	if err != nil {
		return nil, err
	}

	sort.SliceStable(rawFeeds, func(i, j int) bool {
		return rawFeeds[i].Title < rawFeeds[j].Title
	})

	feeds := Feeds{}
	for _, feed := range rawFeeds {
		category := strings.ToLower(feed.Category.Title)
		if _, ok := feeds[category]; !ok {
			feeds[category] = []Feed{}
		}

		feeds[category] = append(feeds[category], Feed{
			Title: feed.Title,
			Feed:  feed.FeedURL,
			Site:  feed.SiteURL,
		})
	}

	return feeds, nil
}
