package eagle

import (
	stripMarkdown "github.com/writeas/go-strip-markdown"
)

const (
	pageSize = 50
)

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
	Date      string   `json:"date" mapstructure:"date"`
}

func sanitizePost(content string) string {
	content = shortcodeRegex.ReplaceAllString(content, "")
	content = stripMarkdown.Strip(content)

	return content
}
