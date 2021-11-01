package eagle

import (
	"sort"
	"strings"

	"github.com/hacdias/eagle/v2/config"
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
