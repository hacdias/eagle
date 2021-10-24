package eagle

import (
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/config"
	"github.com/meilisearch/meilisearch-go"
	"github.com/mitchellh/mapstructure"
)

const (
	searchIndex = "IndexV5"
	searchKey   = "idx"
)

var (
	shortcodeRegex = regexp.MustCompile(`{{<(.*?)>}}`)

	searcheableAttributes = []string{
		"title",
		"tags",
		"content",
	}

	filterableAttributes = []string{
		"section",
		"draft",
		"deleted",
	}

	cropAttributes = []string{
		"content",
	}
)

type MeiliSearch struct {
	meilisearch.ClientInterface
}

func NewMeiliSearch(conf *config.MeiliSearch) (*MeiliSearch, bool, error) {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   conf.Endpoint,
		APIKey: conf.Key,
	})

	ms := &MeiliSearch{
		ClientInterface: client,
	}

	indexes, err := ms.GetAllIndexes()
	if err != nil {
		return nil, false, err
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
			return nil, found, err
		}
	}

	_, err = ms.Index(searchIndex).UpdateSearchableAttributes(&searcheableAttributes)
	if err != nil {
		return nil, found, err
	}

	_, err = ms.Index(searchIndex).UpdateFilterableAttributes(&filterableAttributes)
	if err != nil {
		return nil, found, err
	}

	return ms, found, nil
}

func (ms *MeiliSearch) ResetIndex() error {
	_, err := ms.Index(searchIndex).DeleteAllDocuments()
	return err
}

func (ms *MeiliSearch) Add(entries ...*Entry) error {
	docs := []*SearchEntry{}

	for _, entry := range entries {
		docs = append(docs, &SearchEntry{
			SearchID:  hex.EncodeToString([]byte(entry.ID)),
			ID:        entry.ID,
			Permalink: entry.Permalink,
			Date:      entry.Date(),
			// Searcheable Attributes
			Title:   entry.Metadata.Title,
			Tags:    entry.Metadata.Tags,
			Content: sanitizePost(entry.Content),
			// Filterable Attributes
			Section: entry.Section(),
			Draft:   entry.Metadata.Draft,
			Deleted: entry.Deleted(),
		})
	}

	_, err := ms.Index(searchIndex).UpdateDocuments(docs)
	return err
}

func (ms *MeiliSearch) Remove(entries ...*Entry) error {
	ids := []string{}
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}

	_, err := ms.Index(searchIndex).DeleteDocuments(ids)
	return err
}

func (ms *MeiliSearch) Search(query *SearchQuery, page int) ([]*SearchEntry, error) {
	filters := []string{}

	if query.Deleted != nil {
		filters = append(filters, "(deleted="+strconv.FormatBool(*query.Deleted)+")")
	}

	if query.Draft != nil {
		filters = append(filters, "(draft="+strconv.FormatBool(*query.Draft)+")")
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

	if page != -1 {
		req.Offset = int64(page * pageSize)
		req.Limit = pageSize
	}

	data, err := ms.Index(searchIndex).Search(query.Query, req)
	if err != nil {
		return nil, err
	}

	res := []*SearchEntry{}
	err = mapstructure.Decode(data.Hits, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
