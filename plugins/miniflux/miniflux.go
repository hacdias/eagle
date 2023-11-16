package miniflux

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
	miniflux "miniflux.app/v2/client"
)

func init() {
	server.RegisterPlugin("miniflux", NewMiniflux)
}

type Miniflux struct {
	fs           *core.FS
	client       *miniflux.Client
	jsonFilename string
	opmlFilename string
}

func NewMiniflux(fs *core.FS, config map[string]interface{}) (server.Plugin, error) {
	endpoint := typed.New(config).String("endpoint")
	if endpoint == "" {
		return nil, errors.New("miniflux endpoint missing")
	}

	key := typed.New(config).String("key")
	if key == "" {
		return nil, errors.New("miniflux key missing")
	}

	filename := typed.New(config).String("filename")
	if filename == "" {
		return nil, errors.New("miniflux filename missing")
	}

	return &Miniflux{
		fs:           fs,
		client:       miniflux.New(endpoint, key),
		jsonFilename: filename,
		opmlFilename: typed.New(config).String("opml"),
	}, nil
}

func (mf *Miniflux) GetAction() (string, func() error) {
	return "Update Miniflux Blogroll", mf.Execute
}

func (mf *Miniflux) GetDailyCron() func() error {
	return mf.Execute
}

func (mf *Miniflux) GetWebHandler(utils *server.PluginWebUtilities) (string, http.HandlerFunc) {
	return "", nil
}

func (u *Miniflux) Execute() error {
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
