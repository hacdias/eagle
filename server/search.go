package server

import (
	"net/http"
	"path/filepath"
	"strconv"

	"go.hacdias.com/eagle/core"
)

const (
	searchPath = "/search/"
)

type searchPage struct {
	Entries  core.Entries
	Query    string
	Previous string
	Next     string
}

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	page := 0
	if v := r.URL.Query().Get("page"); v != "" {
		p, _ := strconv.Atoi(v)
		if p >= 0 {
			page = p
		}
	}

	data := &searchPage{
		Query: r.URL.Query().Get("query"),
	}

	if data.Query != "" {
		ee, err := s.meilisearch.Search(int64(page), int64(s.c.Site.Pagination.PagerSize), data.Query)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		rq := r.URL.Query()
		rq.Set("page", strconv.Itoa(page+1))
		if len(ee) == s.c.Site.Pagination.PagerSize {
			data.Next = r.URL.Path + "?" + rq.Encode()
		}

		if page != 0 {
			rq.Set("page", strconv.Itoa(page-1))
			data.Previous = r.URL.Path + "?" + rq.Encode()
		}

		data.Entries = ee
	}

	doc, err := s.getDocument(filepath.Join(r.URL.Path, "index.html"))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.renderDocument(w, r, doc, http.StatusOK, searchTemplate, data)
}
