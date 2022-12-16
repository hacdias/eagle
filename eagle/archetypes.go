package eagle

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/util"
	"github.com/samber/lo"
)

type Archetype func(c *Config, r *http.Request) *Entry

var DefaultArchetypes = map[string]Archetype{
	"default": func(c *Config, r *http.Request) *Entry {
		return &Entry{}
	},
	"article": func(c *Config, r *http.Request) *Entry {
		return &Entry{
			FrontMatter: FrontMatter{
				Title: "Article Title",
				Draft: true,
				Taxonomies: map[string][]string{
					"tags": {"example"},
				},
			},
			Content: "Code is poetry...",
			ID:      NewID("my-article", time.Now()),
		}
	},
	"now": func(c *Config, r *http.Request) *Entry {
		t := time.Now().Local()
		month := t.Format("January")

		return &Entry{
			FrontMatter: FrontMatter{
				Draft:     true,
				Title:     fmt.Sprintf("Recently in %s '%s", month, t.Format("06")),
				Published: t,
				Sections:  []string{"home", "articles"},
				Taxonomies: map[string][]string{
					"tags": {"now"},
				},
			},
			Content: "How was last month?",
			ID:      NewID(fmt.Sprintf("%s-%s", strings.ToLower(month), t.Format("06")), time.Now()),
		}
	},
	"book": func(c *Config, r *http.Request) *Entry {
		name, _ := lo.Coalesce(r.URL.Query().Get("name"), "Name")
		author, _ := lo.Coalesce(r.URL.Query().Get("author"), "Author")
		publisher, _ := lo.Coalesce(r.URL.Query().Get("publisher"), "Publisher")
		isbn, _ := lo.Coalesce(r.URL.Query().Get("isbn"), "ISBN")
		pages, _ := lo.Coalesce(r.URL.Query().Get("pages"), "PAGES")

		date := time.Now().Local()
		return &Entry{
			ID: NewID(util.Slugify(name), time.Now()),
			FrontMatter: FrontMatter{
				Published:   date,
				Description: fmt.Sprintf("%s by %s (ISBN: %s)", name, author, isbn),
				Sections:    []string{"books"},
				Properties: map[string]interface{}{
					"read-of": map[string]interface{}{
						"properties": map[string]interface{}{
							"author":    author,
							"name":      name,
							"pages":     pages,
							"publisher": publisher,
							"uid":       fmt.Sprintf("isbn:%s", isbn),
						},
						"type": "h-cite",
					},
				},
			},
		}
	},
}
