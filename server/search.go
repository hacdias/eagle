package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/porter"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/hacdias/eagle/core"
)

const (
	searchPath = "/search/"
)

func (s *Server) initIndex() error {
	mapping := bleve.NewIndexMapping()
	err := mapping.AddCustomAnalyzer("en-with-stop-words", map[string]interface{}{
		"type":      custom.Name,
		"tokenizer": unicode.Name,
		"token_filters": []string{
			en.PossessiveName,
			lowercase.Name,
			porter.Name,
		},
	})
	if err != nil {
		return err
	}

	titleField := bleve.NewTextFieldMapping()
	titleField.Analyzer = "en-with-stop-words"

	contentField := bleve.NewTextFieldMapping()
	contentField.Analyzer = "en"

	page := bleve.NewDocumentStaticMapping()
	page.DefaultAnalyzer = "en-with-stop-words" // https://github.com/blevesearch/bleve/issues/1835
	page.AddFieldMappingsAt("title", titleField)
	page.AddFieldMappingsAt("content", contentField)

	mapping.DefaultMapping = page

	index, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return fmt.Errorf("could not create index: %w", err)
	}

	s.index = index
	return nil
}

func (s *Server) indexAdd(ee ...*core.Entry) error {
	ss := time.Now()
	b := s.index.NewBatch()
	for _, e := range ee {
		err := b.Index(e.ID, map[string]interface{}{
			"title":   e.Title,
			"tags":    e.Tags,
			"content": e.TextContent(),
		})
		if err != nil {
			return fmt.Errorf("could not add: %w", err)
		}
	}
	err := s.index.Batch(b)
	if err != nil {
		return fmt.Errorf("could not batch: %w", err)
	}

	fmt.Println(s.index.Fields())
	s.log.Infof("bleve update took %dms", time.Since(ss).Milliseconds())
	return nil
}

func (s *Server) indexSearch(page int, query string) (core.Entries, error) {
	from := 0
	if page != -1 {
		from = page * s.c.Pagination
	}

	request := bleve.NewSearchRequestOptions(
		bleve.NewQueryStringQuery(query),
		s.c.Pagination, from, false,
	)

	res, err := s.index.Search(request)
	if err != nil {
		return nil, err
	}

	entries := core.Entries{}
	for _, hit := range res.Hits {
		entry, err := s.fs.GetEntry(hit.ID)
		if err != nil {
			if os.IsNotExist(err) {
				_ = s.index.Delete(hit.ID)
			} else {
				return nil, err
			}
		} else {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	doc, err := s.getTemplateDocument(r.URL.Path)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	page := 0
	if v := r.URL.Query().Get("page"); v != "" {
		p, _ := strconv.Atoi(v)
		if p >= 0 {
			page = p
		}
	}

	query := r.URL.Query().Get("query")
	if query != "" {
		entries, err := s.indexSearch(page, query)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		doc.Find("#eagle-search-input").SetAttr("value", query)

		noResultsNode := doc.Find("eagle-no-search-results")
		if len(entries) == 0 {
			noResultsNode.ReplaceWithSelection(noResultsNode.Children())
		} else {
			noResultsNode.Empty()
		}

		resultsNode := doc.Find("eagle-search-results")
		resultTemplate := doc.Find("eagle-search-result").Children()
		paginationNode := doc.Find("eagle-search-pagination").Children()
		resultsNode.Empty()

		for _, e := range entries {
			node := resultTemplate.Clone()

			title := e.Title
			if title == "" {
				title = e.Description
			}
			if title == "" {
				title = "Untitled Post"
			}

			content := e.TextContent()
			if len(content) > 300 {
				content = content[0:300]
				content = strings.TrimSpace(content) + "…"
			}

			node.Find("entry-title").ReplaceWithHtml(title)
			node.Find("entry-content").ReplaceWithHtml(content)
			node.Find(".entry-link").SetAttr("href", e.Permalink)

			resultsNode.AppendSelection(node)
		}

		rq := r.URL.Query()

		if len(entries) == 0 {
			paginationNode.Find(".eagle-next").Remove()
		} else {
			rq.Set("page", strconv.Itoa(page+1))
			paginationNode.Find(".eagle-next").SetAttr("href", "?"+rq.Encode())
		}

		if page == 0 {
			paginationNode.Find(".eagle-prev").Remove()
		} else {
			rq.Set("page", strconv.Itoa(page-1))
			paginationNode.Find(".eagle-prev").SetAttr("href", "?"+rq.Encode())
		}

		resultsNode.AppendSelection(paginationNode)
		resultsNode.ReplaceWithSelection(resultsNode.Children())
	}

	s.serveDocument(w, r, doc, http.StatusOK)
}
