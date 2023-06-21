package core

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

type Archetype func(c *Config, r *http.Request) *Entry

var DefaultArchetypes = map[string]Archetype{
	"default": func(c *Config, r *http.Request) *Entry {
		return &Entry{
			FrontMatter: FrontMatter{
				Title: "Article Title",
				Draft: true,
				Tags:  []string{"example"},
			},
			Content: "Code is poetry...",
			ID:      newID("articles", "my-article"),
		}
	},
	"now": func(c *Config, r *http.Request) *Entry {
		t := time.Now().Local()
		month := t.Format("January")

		return &Entry{
			FrontMatter: FrontMatter{
				Draft: true,
				Title: fmt.Sprintf("Recently in %s '%s", month, t.Format("06")),
				Date:  t,
				Tags:  []string{"now"},
			},
			Content: "How was last month?",
			ID:      newID("articles", fmt.Sprintf("%s-%s", strings.ToLower(month), t.Format("06"))),
		}
	},
	"book": func(c *Config, r *http.Request) *Entry {
		name, _ := lo.Coalesce(r.URL.Query().Get("name"), "Name")
		author, _ := lo.Coalesce(r.URL.Query().Get("author"), "Author")
		publisher, _ := lo.Coalesce(r.URL.Query().Get("publisher"), "Publisher")
		isbn, _ := lo.Coalesce(r.URL.Query().Get("isbn"), "ISBN")
		pagesStr, _ := lo.Coalesce(r.URL.Query().Get("pages"), "PAGES")
		pages, _ := strconv.Atoi(pagesStr)

		date := time.Now().Local()
		return &Entry{
			ID: newID("readings", slugify(name)),
			FrontMatter: FrontMatter{
				Title: fmt.Sprintf("Read \"%s\" by %s", name, author),
				Date:  date,
				Read: &Read{
					Author:    author,
					Name:      name,
					Pages:     pages,
					Publisher: publisher,
					UID:       fmt.Sprintf("isbn:%s", isbn),
				},
			},
		}
	},
}
