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

var (
	_ server.ActionPlugin  = &Miniflux{}
	_ server.CronPlugin    = &Miniflux{}
	_ server.HandlerPlugin = &Miniflux{}
)

func init() {
	server.RegisterPlugin("miniflux", NewMiniflux)
}

type Miniflux struct {
	core             *core.Core
	client           *miniflux.Client
	jsonFilename     string
	opmlFilename     string
	redirectLocation string
}

func NewMiniflux(co *core.Core, config map[string]interface{}) (server.Plugin, error) {
	endpoint := typed.New(config).String("endpoint")
	if endpoint == "" {
		return nil, errors.New("miniflux endpoint missing")
	}

	key := typed.New(config).String("key")
	if key == "" {
		return nil, errors.New("miniflux key missing")
	}

	jsonFilename := typed.New(config).String("filename")
	if jsonFilename == "" {
		return nil, errors.New("miniflux filename missing")
	}

	opmlFilename := typed.New(config).String("opml")
	redirectLocation := ""
	if strings.HasPrefix(opmlFilename, core.ContentDirectory) {
		redirectLocation = strings.TrimPrefix(opmlFilename, core.ContentDirectory)
	}

	return &Miniflux{
		core:             co,
		client:           miniflux.New(endpoint, key),
		jsonFilename:     jsonFilename,
		opmlFilename:     opmlFilename,
		redirectLocation: redirectLocation,
	}, nil
}

func (mf *Miniflux) ActionName() string {
	return "Update Miniflux Blogroll"
}

func (u *Miniflux) Action() error {
	newFeeds, err := u.fetch()
	if err != nil {
		return err
	}

	var oldFeeds map[string][]feed
	err = u.core.ReadJSON(u.jsonFilename, &oldFeeds)
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

	return u.core.WriteFiles(files, "blogroll: synchronize with miniflux")
}

func (mf *Miniflux) DailyCron() error {
	return mf.Action()
}

func (mf *Miniflux) HandlerRoute() string {
	return wellKnownRecommendationsPath
}

func (mf *Miniflux) Handler(w http.ResponseWriter, r *http.Request, utils *server.PluginWebUtilities) {
	if mf.redirectLocation != "" {
		http.Redirect(w, r, mf.redirectLocation, http.StatusFound)
	} else {
		utils.ErrorHTML(w, r, http.StatusNotFound, nil)
	}
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

const wellKnownRecommendationsPath = "/.well-known/recommendations.opml"
