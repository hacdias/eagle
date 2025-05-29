package linkding

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/karlseguin/typed"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

var (
	_ server.ActionPlugin = &Linkding{}
	_ server.CronPlugin   = &Linkding{}
)

func init() {
	server.RegisterPlugin("linkding", NewLinkding)
}

type Linkding struct {
	core       *core.Core
	httpClient *http.Client
	endpoint   string
	key        string
	filename   string
}

func NewLinkding(co *core.Core, config map[string]any) (server.Plugin, error) {
	endpoint := typed.New(config).String("endpoint")
	if endpoint == "" {
		return nil, errors.New("linkding endpoint missing")
	}

	key := typed.New(config).String("key")
	if key == "" {
		return nil, errors.New("linkding key missing")
	}

	filename := typed.New(config).String("filename")
	if filename == "" {
		return nil, errors.New("linkding filename missing")
	}

	return &Linkding{
		core: co,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		endpoint: strings.TrimSuffix(endpoint, "/"),
		key:      key,
		filename: filename,
	}, nil
}

func (ld *Linkding) ActionName() string {
	return "Update Linkding Bookmarks"
}

func (ld *Linkding) Action() error {
	newBookmarks, err := ld.fetch()
	if err != nil {
		return err
	}

	newBookmarksBytes, err := json.MarshalIndent(newBookmarks, "", "  ")
	if err != nil {
		return err
	}

	var oldBookmarks []bookmark
	err = ld.core.ReadJSON(ld.filename, &oldBookmarks)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	oldBookmarksBytes, err := json.MarshalIndent(oldBookmarks, "", "  ")
	if err != nil {
		return err
	}

	// NOTE: reflect.DeepEqual cannot be used here because [time.Time] representations
	// of the same time might not be the same, and therefore yielding false positives.
	if bytes.Equal(oldBookmarksBytes, newBookmarksBytes) {
		return nil
	}

	return ld.core.WriteFile(ld.filename, newBookmarksBytes, "bookmarks: synchronize with linkding")
}

func (ld *Linkding) DailyCron() error {
	return ld.Action()
}

type bookmark struct {
	URL         string    `json:"url,omitempty"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Date        time.Time `json:"date,omitempty"`
}

type result struct {
	ID                 int       `json:"id,omitempty"`
	URL                string    `json:"url,omitempty"`
	Title              string    `json:"title,omitempty"`
	Description        string    `json:"description,omitempty"`
	Notes              string    `json:"notes,omitempty"`
	WebsiteTitle       string    `json:"website_title,omitempty"`
	WebsiteDescription string    `json:"website_description,omitempty"`
	IsArchived         bool      `json:"is_archived,omitempty"`
	Unread             bool      `json:"unread,omitempty"`
	Shared             bool      `json:"shared,omitempty"`
	TagNames           []string  `json:"tag_names,omitempty"`
	DateAdded          time.Time `json:"date_added,omitempty"`
	DateModified       time.Time `json:"date_modified,omitempty"`
}

type results struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous any    `json:"previous"`
	Results  []result
}

func (ld *Linkding) fetch() ([]bookmark, error) {
	var bookmarks []bookmark
	for p := 1; ; p++ {
		newBookmarks, err := ld.fetchPage(p)
		if err != nil {
			return nil, err
		}

		bookmarks = append(bookmarks, newBookmarks...)
		if len(newBookmarks) == 0 {
			break
		}
	}

	sort.SliceStable(bookmarks, func(i, j int) bool {
		return bookmarks[i].Date.After(bookmarks[j].Date)
	})

	return bookmarks, nil
}

func (ld *Linkding) fetchPage(page int) ([]bookmark, error) {
	q := url.Values{}
	q.Set("limit", "100")
	q.Set("offset", strconv.Itoa((page-1)*100))

	req, err := http.NewRequest(http.MethodGet, ld.endpoint+"/api/bookmarks/?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Token "+ld.key)

	res, err := ld.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var data *results
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	var bookmarks []bookmark

	for _, b := range data.Results {
		title, _ := lo.Coalesce(b.Title, b.WebsiteTitle)
		description, _ := lo.Coalesce(b.Notes, b.Description)

		bookmarks = append(bookmarks, bookmark{
			URL:         b.URL,
			Title:       title,
			Description: description,
			Tags:        b.TagNames,
			Date:        b.DateAdded,
		})
	}

	return bookmarks, nil
}
