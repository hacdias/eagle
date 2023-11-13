package linkding

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
)

type Linkding struct {
	fs           *core.FS
	httpClient   *http.Client
	endpoint     string
	key          string
	jsonFilename string
}

func NewLinkding(c *core.Linkding, fs *core.FS) *Linkding {
	return &Linkding{
		fs: fs,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		endpoint:     strings.TrimSuffix(c.Endpoint, "/"),
		key:          c.Key,
		jsonFilename: c.JSON,
	}
}

func (u *Linkding) Synchronize() error {
	newBookmarks, err := u.fetch()
	if err != nil {
		return err
	}

	var oldBookmarks []bookmark
	err = u.fs.ReadJSON(u.jsonFilename, &oldBookmarks)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if reflect.DeepEqual(oldBookmarks, newBookmarks) {
		return nil
	}

	return u.fs.WriteJSON(u.jsonFilename, newBookmarks, "bookmarks: synchronize with linkding")
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
