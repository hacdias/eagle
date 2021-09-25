package eagle

import (
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/meilisearch/meilisearch-go"
	stripMarkdown "github.com/writeas/go-strip-markdown"
)

const (
	searchIndex = "IndexV4"
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
	docs := []interface{}{}

	for _, entry := range entries {
		if entry.Metadata.ExpiryDate.After(time.Now()) {
			continue
		}

		cleanID := strings.TrimPrefix(entry.ID, "/")
		cleanID = strings.TrimSuffix(cleanID, "/")

		section := ""
		if strings.Count(cleanID, "/") >= 1 {
			section = strings.Split(cleanID, "/")[0]
		}

		docs = append(docs, map[string]interface{}{
			searchKey: hex.EncodeToString([]byte(entry.ID)),
			"id":      entry.ID,
			// Searcheable Attributes
			"title":   entry.Metadata.Title,
			"tags":    entry.Metadata.Tags,
			"content": sanitizePost(entry.Content),
			// Filterable Attributes
			"section": section,
			"draft":   entry.Metadata.Draft,
			// Other Attributes
			"date": entry.Metadata.Date,
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

func (ms *MeiliSearch) Search(query *SearchQuery, page int) ([]interface{}, error) {
	sectionsCond := []string{}

	if query.Sections != nil {
		for _, s := range query.Sections {
			sectionsCond = append(sectionsCond, "section=\""+s+"\"")
		}
	}

	filter := ""
	if len(sectionsCond) > 0 {
		filter = "(" + strings.Join(sectionsCond, " OR ") + ") AND "
	}

	filter = filter + "(draft=" + strconv.FormatBool(query.Draft) + ")"

	req := &meilisearch.SearchRequest{
		Filter:           filter,
		AttributesToCrop: cropAttributes,
		CropLength:       200,
	}

	if page != -1 {
		req.Offset = int64(page * 20)
		req.Limit = 20
	}

	res, err := ms.Index(searchIndex).Search(query.Query, req)

	if err != nil {
		return nil, err
	}

	return res.Hits, nil
}

func sanitizePost(content string) string {
	content = shortcodeRegex.ReplaceAllString(content, "")
	content = stripMarkdown.Strip(content)

	return content
}
