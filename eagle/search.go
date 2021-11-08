package eagle

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/meilisearch/meilisearch-go"
	"github.com/mitchellh/mapstructure"
	stripMarkdown "github.com/writeas/go-strip-markdown"
)

const (
	searchIndex = "IndexV7"
	searchKey   = "idx"
)

type SearchQuery struct {
	Query    string
	Sections []string // If empty, matches all sections
	Tags     []string // If empty, matches all tags
	Year     int
	Month    int
	Day      int
	Page     int
	ByDate   bool
	Draft    bool
	Deleted  bool
	Private  bool
}

type SearchEntry struct {
	SearchID string `json:"idx" mapstructure:"idx"`

	Year  int `json:"year" mapstructure:"year"`
	Month int `json:"month" mapstructure:"month"`
	Day   int `json:"day" mapstructure:"day"`

	ID      string   `json:"id" mapstructure:"id"`
	Title   string   `json:"title" mapstructure:"title"`
	Tags    []string `json:"tags" mapstructure:"tags"`
	Content string   `json:"content" mapstructure:"content"`
	Section string   `json:"section" mapstructure:"section"`
	Date    int64    `json:"date" mapstructure:"date"`
	Draft   bool     `json:"draft" mapstructure:"draft"`
	Deleted bool     `json:"deleted" mapstructure:"deleted"`
	Private bool     `json:"private" mapstructure:"private"`
}

var (
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
		"year",
		"month",
		"day",
	}

	cropAttributes = []string{
		"content",
	}

	sortableAttributes = []string{
		"date",
	}
)

func sanitizePost(content string) string {
	content = stripMarkdown.Strip(content)
	return content
}

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
		docs = append(docs, &SearchEntry{
			SearchID: hex.EncodeToString([]byte(entry.ID)),
			ID:       entry.ID,
			Date:     entry.Published.Unix(),
			// Searcheable Attributes
			Title:   entry.Title,
			Content: sanitizePost(entry.Content),
			// Filterable Attributes
			Year:    entry.Published.Year(),
			Month:   int(entry.Published.Month()),
			Day:     entry.Published.Day(),
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

func (e *Eagle) Search(query *SearchQuery) ([]*Entry, error) {
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

	if query.Year > 0 {
		filters = append(filters, "(year="+strconv.Itoa(query.Year)+")")
	}

	if query.Month > 0 {
		filters = append(filters, "(month="+strconv.Itoa(query.Month)+")")
	}

	if query.Day > 0 {
		filters = append(filters, "(day="+strconv.Itoa(query.Day)+")")
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

	if query.Page != -1 {
		req.Offset = int64(query.Page * e.Config.Site.Paginate)
		req.Limit = int64(e.Config.Site.Paginate)
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
		entry, err := e.GetEntry(se.ID)
		if err != nil {
			return nil, err
		}

		entries[i] = entry
	}

	return entries, nil
}
