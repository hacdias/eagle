package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/indexer"
)

func (s *Server) getPagination(r *http.Request) indexer.Pagination {
	opts := indexer.Pagination{
		Limit: s.c.Site.Pagination,
	}

	if v := chi.URLParam(r, "page"); v != "" {
		p, _ := strconv.Atoi(v)
		if p >= 0 {
			opts.Page = p
		}
	}

	return opts
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	if query == "" {
		s.generalHandler(w, r)
		return
	}

	// TODO: pagination like Hugo's own pagination.
	// Search file should have an element <search-results> parse /search/index.html
	// Replace <search-results> with values.

	options := &indexer.Query{
		Pagination:   s.getPagination(r),
		WithDrafts:   s.isLoggedIn(r),
		WithDeleted:  s.isLoggedIn(r),
		WithUnlisted: s.isLoggedIn(r),
	}

	entries, err := s.i.GetSearch(options, query)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO
	fmt.Println(entries)

	doc, err := s.getTemplateDocument(r.URL.Path) // TODO: change when pages
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc.Find("#eagle-search-input").SetAttr("value", query)
	doc.Find("search-results").ReplaceWithHtml("RESULTS HERE")

	s.serveDocument(w, r, doc, http.StatusOK)
}
