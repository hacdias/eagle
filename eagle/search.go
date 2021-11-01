package eagle

import (
	"time"

	stripMarkdown "github.com/writeas/go-strip-markdown"
)

const (
	pageSize = 50
)

type NewSearchQuery struct {
	Query    string
	Sections []string // if empty, matches all sections
	Tags     []string // if empty, matches all tags
	ByDate   bool
	Before   time.Time
	Draft    *bool
	Deleted  *bool
	Private  *bool
}

type SearchQuery struct {
	Query    string
	Sections []string // if empty, matches all sections
	ByDate   bool
	Draft    *bool
	Deleted  *bool
}

type SearchIndex interface {
	ResetIndex() error
	Add(entries ...*Entry) error
	Remove(entries ...*Entry) error
	Search(query *SearchQuery, page int) ([]*SearchEntry, error)
}

type SearchEntry struct {
	// SearchID is for Meilisearch. See searchKey.
	SearchID string `json:"idx" mapstructure:"idx"`

	ID        string   `json:"id" mapstructure:"id"`
	Permalink string   `json:"permalink" mapstructure:"permalink"`
	Title     string   `json:"title" mapstructure:"title"`
	Tags      []string `json:"tags" mapstructure:"tags"`
	Content   string   `json:"content" mapstructure:"content"`
	Section   string   `json:"section" mapstructure:"section"`
	Draft     bool     `json:"draft" mapstructure:"draft"`
	Deleted   bool     `json:"deleted" mapstructure:"deleted"`
	Private   bool     `json:"private" mapstructure:"private"`
	Date      string   `json:"date" mapstructure:"date"`
}

func sanitizePost(content string) string {
	content = shortcodeRegex.ReplaceAllString(content, "")
	content = stripMarkdown.Strip(content)

	return content
}

func (e *Eagle) Search(query *SearchQuery, page int) ([]*SearchEntry, error) {
	if e.search == nil {
		return []*SearchEntry{}, nil
	}

	return e.search.Search(query, page)
}

func (e *Eagle) RebuildIndex() error {
	if e.search == nil {
		return nil
	}

	err := e.search.ResetIndex()
	if err != nil {
		return err
	}

	entries, err := e.GetAllEntries()
	if err != nil {
		return err
	}

	return e.search.Add(entries...)
}
