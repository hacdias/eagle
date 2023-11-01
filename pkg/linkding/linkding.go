package linkding

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

type Linkding struct {
	endpoint   string
	key        string
	httpClient *http.Client
}

func NewLinkding(endpoint, key string) *Linkding {
	return &Linkding{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		key:      key,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (ld *Linkding) Fetch() ([]Bookmark, error) {
	var bookmarks []Bookmark
	for p := 1; ; p++ {
		newBookmarks, err := ld.fetch(p)
		if err != nil {
			return nil, err
		}

		bookmarks = append(bookmarks, newBookmarks...)
		if len(newBookmarks) == 0 {
			break
		}
	}

	sort.Slice(bookmarks, func(i, j int) bool {
		return bookmarks[i].Date.After(bookmarks[j].Date)
	})

	return bookmarks, nil
}

func (ld *Linkding) fetch(page int) ([]Bookmark, error) {
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

	var bookmarks []Bookmark

	for _, b := range data.Results {
		title, _ := lo.Coalesce(b.Title, b.WebsiteTitle)
		description, _ := lo.Coalesce(b.Notes, b.Description)

		bookmarks = append(bookmarks, Bookmark{
			URL:         b.URL,
			Title:       title,
			Description: description,
			Tags:        b.TagNames,
			Date:        b.DateAdded,
		})
	}

	return bookmarks, nil
}

type Bookmark struct {
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
