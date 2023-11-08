package meilisearch

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/karlseguin/typed"
	"github.com/meilisearch/meilisearch-go"
	"go.hacdias.com/eagle/core"
)

const (
	searchIndex = "eagle-v1"
	searchKey   = "idx"
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

type MeiliSearch struct {
	client meilisearch.ClientInterface
	fs     *core.FS
}

func NewMeiliSearch(host, key string, fs *core.FS) (*MeiliSearch, error) {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   host,
		APIKey: key,
	})

	indexes, err := client.GetIndexes(nil)
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

	return &MeiliSearch{
		client: client,
		fs:     fs,
	}, nil
}

func (ms *MeiliSearch) ResetIndex() error {
	_, err := ms.client.Index(searchIndex).DeleteAllDocuments()
	return err
}

func (ms *MeiliSearch) Add(ee ...*core.Entry) error {
	docs := []interface{}{}

	for _, e := range ee {
		if e.Deleted() || e.Draft {
			continue
		}

		docs = append(docs, map[string]interface{}{
			searchKey: hex.EncodeToString([]byte(e.ID)),
			"id":      e.ID,
			"title":   e.Title,
			"tags":    e.Tags,
			"content": e.TextContent(),
		})
	}

	_, err := ms.client.Index(searchIndex).UpdateDocuments(docs)
	return err
}

func (ms *MeiliSearch) Remove(ids ...string) error {
	_, err := ms.client.Index(searchIndex).DeleteDocuments(ids)
	return err
}

func (ms *MeiliSearch) Search(page, limit int64, query string) (core.Entries, error) {
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
		m, ok := hit.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot convert hit in map[string]interface{}: %q", hit)
		}
		id, ok := typed.Typed(m).StringIf("id")
		if !ok {
			return nil, errors.New("hit does not contain id field")
		}

		entry, err := ms.fs.GetEntry(id)
		if err != nil {
			if os.IsNotExist(err) {
				_ = ms.Remove(id)
			} else {
				return nil, err
			}
		} else {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
