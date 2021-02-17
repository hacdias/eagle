package eagle

import (
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/meilisearch/meilisearch-go"
	stripmd "github.com/writeas/go-strip-markdown"
)

const meiliSearchIndex = "website3"
const meiliSearchKey = "id"

var shortCodesRegex = regexp.MustCompile(`{{<(.*?)>}}`)

type MeiliSearch struct {
	meilisearch.ClientInterface
}

func NewMeiliSearch(conf *config.MeiliSearch) (*MeiliSearch, bool, error) {
	client := meilisearch.NewClient(meilisearch.Config{
		Host:   conf.Endpoint,
		APIKey: conf.Key,
	})

	ms := &MeiliSearch{
		ClientInterface: client,
	}

	indexes, err := ms.Indexes().List()
	if err != nil {
		return nil, false, err
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
			return nil, found, err
		}
	}

	_, err = ms.Settings(meiliSearchIndex).UpdateSearchableAttributes([]string{
		"title",
		"tags",
		"content",
	})
	if err != nil {
		return nil, found, err
	}

	return ms, found, nil
}

func (ms *MeiliSearch) ResetIndex() error {
	_, err := ms.Documents(meiliSearchIndex).DeleteAllDocuments()
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
			meiliSearchKey: hex.EncodeToString([]byte(entry.ID)),
			"title":        entry.Metadata.Title,
			"date":         entry.Metadata.Date,
			"section":      section,
			"content":      sanitizePost(entry.Content),
			"tags":         entry.Metadata.Tags,
			"draft":        entry.Metadata.Draft,
		})
	}

	_, err := ms.Documents(meiliSearchIndex).AddOrUpdate(docs)
	return err
}

func (ms *MeiliSearch) Remove(entries ...*Entry) error {
	ids := []string{}
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}

	_, err := ms.Documents(meiliSearchIndex).Deletes(ids)
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

	req := meilisearch.SearchRequest{
		Query:            query.Query,
		Filters:          filter,
		AttributesToCrop: []string{"content"},
		CropLength:       200,
	}

	if page != -1 {
		req.Offset = int64(page * 20)
		req.Limit = 20
	}

	res, err := ms.ClientInterface.Search(meiliSearchIndex).Search(req)

	if err != nil {
		return nil, err
	}

	return res.Hits, nil
}

func sanitizePost(content string) string {
	content = shortCodesRegex.ReplaceAllString(content, "")
	content = stripmd.Strip(content)

	return content
}
