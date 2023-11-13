package miniflux

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sort"
	"strings"

	"go.hacdias.com/eagle/core"
	miniflux "miniflux.app/v2/client"
)

type Miniflux struct {
	fs           *core.FS
	client       *miniflux.Client
	jsonFilename string
	opmlFilename string
}

func NewMiniflux(c *core.Miniflux, fs *core.FS) *Miniflux {
	return &Miniflux{
		fs:           fs,
		client:       miniflux.New(c.Endpoint, c.Key),
		jsonFilename: c.JSON,
		opmlFilename: c.OPML,
	}
}

func (u *Miniflux) Synchronize() error {
	if u.jsonFilename == "" {
		return errors.New("miniflux: blogroll updater must have JSON filename set")
	}

	newFeeds, err := u.fetch()
	if err != nil {
		return err
	}

	var oldFeeds map[string][]feed
	err = u.fs.ReadJSON(u.jsonFilename, &oldFeeds)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if reflect.DeepEqual(oldFeeds, newFeeds) {
		return nil
	}

	feedsData, err := json.MarshalIndent(newFeeds, "", "  ")
	if err != nil {
		return err
	}

	files := map[string][]byte{
		u.jsonFilename: feedsData,
	}

	if u.opmlFilename != "" {
		opmlData, err := u.client.Export()
		if err != nil {
			return err
		}

		files[u.opmlFilename] = opmlData
	}

	return u.fs.WriteFiles(files, "blogroll: synchronize with miniflux")
}

type feed struct {
	Title string `json:"title"`
	Site  string `json:"site"`
	Feed  string `json:"feed"`
}

func (u *Miniflux) fetch() (map[string][]feed, error) {
	rawFeeds, err := u.client.Feeds()
	if err != nil {
		return nil, err
	}

	sort.SliceStable(rawFeeds, func(i, j int) bool {
		return rawFeeds[i].Title < rawFeeds[j].Title
	})

	feedsByCategory := map[string][]feed{}
	for _, f := range rawFeeds {
		category := strings.ToLower(f.Category.Title)
		if _, ok := feedsByCategory[category]; !ok {
			feedsByCategory[category] = []feed{}
		}

		feedsByCategory[category] = append(feedsByCategory[category], feed{
			Title: f.Title,
			Feed:  f.FeedURL,
			Site:  f.SiteURL,
		})
	}

	return feedsByCategory, nil
}
