package server

import (
	"fmt"
	"net/http"

	"github.com/hacdias/eagle/indexer"
)

func (s *Server) searchGet(w http.ResponseWriter, r *http.Request) {
	search := &indexer.Search{
		Query: r.URL.Query().Get("query"),
		// Sections: []string{},
	}

	// TODO: pagination like Hugo's own pagination.
	// Search file should have an element <search-results> parse /search/index.html
	// Replace <search-results> with values.

	entries, err := s.i.GetSearch(nil, search)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO
	fmt.Println(entries)
	w.WriteHeader(http.StatusNotImplemented)
}
