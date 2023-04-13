package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/indexer"
)

const (
	deletedPath  = "/deleted/"
	draftsPath   = "/drafts/"
	unlistedPath = "/unlisted/"
	searchPath   = "/search/"
)

func (s *Server) getPagination(r *http.Request) *indexer.Pagination {
	opts := &indexer.Pagination{
		Limit: s.c.Site.Pagination,
	}

	if v := r.URL.Query().Get("page"); v != "" {
		p, _ := strconv.Atoi(v)
		if p >= 0 {
			opts.Page = p
		}
	}

	return opts
}

func (s *Server) draftsGet(w http.ResponseWriter, r *http.Request) {
	entries, err := s.i.GetDrafts(s.getPagination(r))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO
	fmt.Println(entries)
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) unlistedGet(w http.ResponseWriter, r *http.Request) {
	entries, err := s.i.GetUnlisted(s.getPagination(r))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO
	fmt.Println(entries)
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) deletedGet(w http.ResponseWriter, r *http.Request) {
	entries, err := s.i.GetDeleted(s.getPagination(r))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO
	fmt.Println(entries)
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	doc, err := s.getTemplateDocument(searchPath)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	query := r.URL.Query().Get("query")
	if query != "" {
		loggedIn := s.isLoggedIn(r)
		options := &indexer.Query{
			Pagination:   *s.getPagination(r),
			WithDrafts:   loggedIn,
			WithDeleted:  loggedIn,
			WithUnlisted: loggedIn,
		}

		entries, err := s.i.GetSearch(options, query)
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
			rq.Set("page", strconv.Itoa(options.Page+1))
			paginationNode.Find(".eagle-next").SetAttr("href", "?"+rq.Encode())
		}

		if options.Page == 0 {
			paginationNode.Find(".eagle-prev").Remove()
		} else {
			rq.Set("page", strconv.Itoa(options.Page-1))
			paginationNode.Find(".eagle-prev").SetAttr("href", "?"+rq.Encode())
		}

		resultsNode.AppendSelection(paginationNode)
		resultsNode.ReplaceWithSelection(resultsNode.Children())
	}

	s.serveDocument(w, r, doc, http.StatusOK)
}
