package services

/*
import (
	"encoding/hex"
	"strings"

	"github.com/hacdias/eagle/config"
	"github.com/meilisearch/meilisearch-go"
	stripmd "github.com/writeas/go-strip-markdown"
)

const meiliSearchIndex = "posts"
const meiliSearchKey = "id"

type MeiliSearch struct {
	meilisearch.ClientInterface
}

func NewMeiliSearch(c *config.MeiliSearch) (*MeiliSearch, error) {
	client := meilisearch.NewClient(meilisearch.Config{
		Host:   c.Endpoint,
		APIKey: c.Key,
	})

	ms := &MeiliSearch{
		ClientInterface: client,
	}

	indexes, err := ms.Indexes().List()
	if err != nil {
		return nil, err
	}

	found := false
	for _, idx := range indexes {
		if idx.Name == meiliSearchIndex {
			found = true
		}
	}

	if !found {
		_, err := ms.Indexes().Create(meilisearch.CreateIndexRequest{
			UID:        meiliSearchIndex,
			PrimaryKey: meiliSearchKey,
		})

		if err != nil {
			return nil, err
		}
	}

	return ms, nil
}

func (ms *MeiliSearch) Wipe() error {
	_, err := ms.Documents(meiliSearchIndex).DeleteAllDocuments()
	return err
}

func (ms *MeiliSearch) Add(entries ...*HugoEntry) error {
	docs := []interface{}{}

	for _, entry := range entries {
		tags := []string{}
		if t, ok := entry.Metadata.StringsIf("tags"); ok {
			tags = t
		}

		title := ""
		if t, ok := entry.Metadata.StringIf("title"); ok {
			title = t
		}

		date := ""
		if d, ok := entry.Metadata.StringIf("date"); ok {
			date = d
		}

		cleanID := strings.TrimPrefix(entry.ID, "/")
		cleanID = strings.TrimSuffix(cleanID, "/")

		section := ""
		if strings.Count(cleanID, "/") >= 1 {
			section = strings.Split(cleanID, "/")[0]
		}

		docs = append(docs, map[string]interface{}{
			meiliSearchKey: hex.EncodeToString([]byte(entry.ID)),
			"title":        title,
			"date":         date,
			"section":      section,
			"content":      stripmd.Strip(entry.Content),
			"tags":         tags,
		})
	}

	_, err := ms.Documents(meiliSearchIndex).AddOrUpdate(docs)
	return err
}

func (ms *MeiliSearch) Delete(entries ...*HugoEntry) error {
	ids := []string{}
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}

	_, err := ms.Documents(meiliSearchIndex).Deletes(ids)
	return err
}

func (ms *MeiliSearch) Search(query string, filter string, page int) ([]interface{}, error) {
	res, err := ms.ClientInterface.Search(meiliSearchIndex).Search(meilisearch.SearchRequest{
		Query:            query,
		Filters:          filter,
		Offset:           int64(page * 20),
		Limit:            20,
		AttributesToCrop: []string{"content"},
		CropLength:       200,
	})

	if err != nil {
		return nil, err
	}

	return res.Hits, nil
}
*/
