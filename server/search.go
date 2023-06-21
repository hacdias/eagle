package server

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	searchPath = "/search/"
)

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
		entries, err := s.meilisearch.Search(int64(page), int64(s.c.Pagination), query)
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
