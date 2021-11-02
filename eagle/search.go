package eagle

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/mitchellh/mapstructure"
	stripMarkdown "github.com/writeas/go-strip-markdown"
)

const (
	pageSize = 50
)

type SearchQuery struct {
	Query    string
	Sections []string // If empty, matches all sections
	Tags     []string // If empty, matches all tags
	Year     int
	Month    int
	Day      int
	ByDate   bool
	Before   time.Time
	Draft    bool
	Deleted  bool
	Private  bool
}

type SearchIndex interface {
	ResetIndex() error
	Add(entries ...*Entry) error
	Remove(entries ...*Entry) error
	Search(query *SearchQuery, page int) ([]*Entry, error)
}

type SearchEntry struct {
	// SearchID is for Meilisearch. See searchKey.
	SearchID string `json:"idx" mapstructure:"idx"`

	RawFile string `json:"rawFile" mapstructure:"rawFile"`

	ID      string   `json:"id" mapstructure:"id"`
	Title   string   `json:"title" mapstructure:"title"`
	Tags    []string `json:"tags" mapstructure:"tags"`
	Content string   `json:"content" mapstructure:"content"`
	Section string   `json:"section" mapstructure:"section"`
	Date    string   `json:"date" mapstructure:"date"`
	Draft   bool     `json:"draft" mapstructure:"draft"`
	Deleted bool     `json:"deleted" mapstructure:"deleted"`
	Private bool     `json:"private" mapstructure:"private"`
}

func sanitizePost(content string) string {
	content = shortcodeRegex.ReplaceAllString(content, "")
	content = stripMarkdown.Strip(content)

	return content
}

const (
	searchIndex = "IndexV6"
	searchKey   = "idx"
)

var (
	shortcodeRegex = regexp.MustCompile(`{{<(.*?)>}}`)

	searcheableAttributes = []string{
		"title",

		"content",
	}

	filterableAttributes = []string{
		"section",
		"tags",
		"draft",
		"deleted",
		"private",
	}

	cropAttributes = []string{
		"content",
	}

	sortableAttributes = []string{
		"date",
	}
)

func (e *Eagle) setupMeiliSearch() error {
	ms := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   e.Config.MeiliSearch.Endpoint,
		APIKey: e.Config.MeiliSearch.Key,
	})

	indexes, err := ms.GetAllIndexes()
	if err != nil {
		return err
	}

	found := false
	for _, idx := range indexes {
		if idx.UID == searchIndex {
			found = true
		}
	}

	if !found {
		_, err := ms.CreateIndex(&meilisearch.IndexConfig{
			Uid:        searchIndex,
			PrimaryKey: searchKey,
		})

		if err != nil {
			return err
		}
	}

	_, err = ms.Index(searchIndex).UpdateSearchableAttributes(&searcheableAttributes)
	if err != nil {
		return err
	}

	_, err = ms.Index(searchIndex).UpdateFilterableAttributes(&filterableAttributes)
	if err != nil {
		return err
	}

	_, err = ms.Index(searchIndex).UpdateSortableAttributes(&sortableAttributes)
	if err != nil {
		return err
	}

	e.ms = ms

	if !found {
		go func() {
			e.log.Info("building index for the first time")
			err = e.RebuildIndex()
			if err != nil {
				err = fmt.Errorf("could not start meilisearch: %w", err)
				e.log.Error(err)
			}

		}()
	}

	return nil
}

func (e *Eagle) RebuildIndex() error {
	_, err := e.ms.Index(searchIndex).DeleteAllDocuments()
	if err != nil {
		return err
	}

	entries, err := e.GetAllEntries()
	if err != nil {
		return err
	}

	return e.IndexAdd(entries...)
}

func (e *Eagle) IndexAdd(entries ...*Entry) error {
	docs := []*SearchEntry{}

	for _, entry := range entries {
		raw, err := entry.String()
		if err != nil {
			return err
		}
		docs = append(docs, &SearchEntry{
			SearchID: hex.EncodeToString([]byte(entry.ID)),
			ID:       entry.ID,
			RawFile:  raw,
			Date:     entry.Published.Format(time.RFC3339),
			// Searcheable Attributes
			Title:   entry.Title,
			Content: sanitizePost(entry.Content),
			// Filterable Attributes
			Tags:    entry.Tags(),
			Section: entry.Section,
			Draft:   entry.Draft,
			Deleted: entry.Deleted,
			Private: entry.Private,
		})
	}

	_, err := e.ms.Index(searchIndex).UpdateDocuments(docs)
	return err
}

func (e *Eagle) Search(query *SearchQuery, page int) ([]*Entry, error) {
	filters := []string{}

	if !query.Deleted {
		filters = append(filters, "(deleted=false)")
	}

	if !query.Private {
		filters = append(filters, "(private=false)")
	}

	if !query.Draft {
		filters = append(filters, "(draft=false)")
	}

	sections := []string{}
	if query.Sections != nil {
		for _, s := range query.Sections {
			sections = append(sections, "section=\""+s+"\"")
		}
	}

	if len(sections) > 0 {
		filters = append(filters, "("+strings.Join(sections, " OR ")+")")
	}

	tags := []string{}
	if query.Tags != nil {
		for _, s := range query.Tags {
			tags = append(tags, "tags=\""+s+"\"")
		}
	}

	if len(tags) > 0 {
		filters = append(filters, "("+strings.Join(tags, " OR ")+")")
	}

	var filter interface{}
	if len(filters) > 0 {
		filter = strings.Join(filters, " AND ")
	} else {
		filter = nil
	}

	req := &meilisearch.SearchRequest{
		Filter:           filter,
		AttributesToCrop: cropAttributes,
		CropLength:       200,
	}

	if query.ByDate {
		req.Sort = []string{"date:desc"}
	}

	if page != -1 {
		req.Offset = int64(page * pageSize)
		req.Limit = pageSize
	}

	data, err := e.ms.Index(searchIndex).Search(query.Query, req)
	if err != nil {
		return nil, err
	}

	res := []*SearchEntry{}
	err = mapstructure.Decode(data.Hits, &res)
	if err != nil {
		return nil, err
	}

	entries := make([]*Entry, len(res))

	for i, se := range res {
		entry, err := e.ParseEntry(se.ID, se.RawFile)
		if err != nil {
			return nil, err
		}

		entries[i] = entry
	}

	return entries, nil
}
