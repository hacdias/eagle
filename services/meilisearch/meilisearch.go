package meilisearch

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/meilisearch/meilisearch-go"
	"go.hacdias.com/eagle/core"
)

const (
	searchIndex = "eagle-v1"
	searchKey   = "idx"

	// maxTags is the maximum number of tags returned by [Meilisearch.Tags].
	maxTags = 1000
)

var (
	searcheableAttributes = []string{
		"title",
		"tags",
		"content",
	}
)

type Pagination struct {
	Page  int
	Limit int
}

type Meilisearch struct {
	client meilisearch.ServiceManager
	core   *core.Core
}

func NewMeilisearch(host, key string, co *core.Core) (*Meilisearch, error) {
	client := meilisearch.New(host, meilisearch.WithAPIKey(key))

	indexes, err := client.ListIndexes(nil)
	if err != nil {
		return nil, err
	}

	found := false
	for _, idx := range indexes.Results {
		if idx.UID == searchIndex {
			found = true
		}
	}

	if !found {
		_, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        searchIndex,
			PrimaryKey: searchKey,
		})

		if err != nil {
			return nil, err
		}
	}

	_, err = client.Index(searchIndex).UpdateSearchableAttributes(&searcheableAttributes)
	if err != nil {
		return nil, err
	}

	// Tags must be filterable to be used as a facet.
	_, err = client.Index(searchIndex).UpdateFilterableAttributes(&[]any{"tags"})
	if err != nil {
		return nil, err
	}

	_, err = client.Index(searchIndex).UpdateFaceting(&meilisearch.Faceting{
		MaxValuesPerFacet: maxTags,
	})
	if err != nil {
		return nil, err
	}

	return &Meilisearch{
		client: client,
		core:   co,
	}, nil
}

func (ms *Meilisearch) ResetIndex() error {
	_, err := ms.client.Index(searchIndex).DeleteAllDocuments(nil)
	return err
}

// indexable returns whether the entry can be publicly searched.
func indexable(e *core.Entry) bool {
	return !e.Deleted() && !e.Draft && !e.Future() && !e.NoIndex
}

func (ms *Meilisearch) Add(ee ...*core.Entry) error {
	docs := []any{}
	removed := []string{}

	for _, e := range ee {
		if !indexable(e) {
			removed = append(removed, e.ID)
			continue
		}

		docs = append(docs, map[string]any{
			searchKey: hex.EncodeToString([]byte(e.ID)),
			"id":      e.ID,
			"title":   e.Title,
			"tags":    e.Tags,
			"content": e.TextContent(),
		})
	}

	if len(removed) > 0 {
		err := ms.Remove(removed...)
		if err != nil {
			return err
		}
	}

	_, err := ms.client.Index(searchIndex).UpdateDocuments(docs, nil)
	return err
}

func (ms *Meilisearch) Remove(ids ...string) error {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = hex.EncodeToString([]byte(id))
	}

	_, err := ms.client.Index(searchIndex).DeleteDocuments(keys, nil)
	return err
}

// Tags returns the number of indexed entries per tag.
func (ms *Meilisearch) Tags() (map[string]int64, error) {
	res, err := ms.client.Index(searchIndex).Search("", &meilisearch.SearchRequest{
		// Only the facet distribution is needed. A limit of 0 would be omitted.
		Limit:                1,
		AttributesToRetrieve: []string{searchKey},
		Facets:               []string{"tags"},
	})
	if err != nil {
		return nil, err
	}

	// Tags are omitted from the distribution if no entry has them.
	facets := struct {
		Tags map[string]int64 `json:"tags"`
	}{Tags: map[string]int64{}}
	err = json.Unmarshal(res.FacetDistribution, &facets)
	if err != nil {
		return nil, err
	}

	return facets.Tags, nil
}

func (ms *Meilisearch) Search(page, limit int64, query string) (core.Entries, error) {
	req := &meilisearch.SearchRequest{
		CropLength: 200,
		Limit:      limit,
	}

	if page != -1 {
		req.Offset = page * limit
	}

	res, err := ms.client.Index(searchIndex).Search(query, req)

	if err != nil {
		return nil, err
	}

	entries := core.Entries{}
	for _, hit := range res.Hits {
		m := struct {
			ID string `json:"id"`
		}{}
		if err := hit.DecodeInto(&m); err != nil {
			return nil, fmt.Errorf("cannot convert hit in map[string]any: %q", hit)
		}

		entry, err := ms.core.GetEntry(m.ID)
		if err != nil {
			if os.IsNotExist(err) {
				_ = ms.Remove(m.ID)
			} else {
				return nil, err
			}
		} else if !indexable(entry) {
			// The entry may have been unpublished, or expired, since indexed.
			_ = ms.Remove(m.ID)
		} else {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
