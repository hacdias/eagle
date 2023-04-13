package server

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/indexer"
	"github.com/hacdias/eagle/pkg/contenttype"
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

	f, err := s.staticFs.ReadFile("/search/index.html")
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(f))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc.Find("#eagle-search-input").SetAttr("value", query)
	doc.Find("search-results").ReplaceWithHtml("RESULTS HERE")

	html, err := doc.Html()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", contenttype.HTMLUTF8)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(html))
	if err != nil {
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
	}
}
