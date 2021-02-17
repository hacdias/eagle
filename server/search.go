package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/eagle"
)

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var err error
	page := 0
	if p := r.URL.Query().Get("p"); p != "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			s.serveError(w, http.StatusBadRequest, err)
			return
		}
	}

	// Parse sections
	sectionsQuery := strings.TrimSpace(r.URL.Query().Get("s"))
	sectionsList := strings.Split(sectionsQuery, ",")
	parsedSections := []string{}

	for _, s := range sectionsList {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		parsedSections = append(parsedSections, s)
	}

	// Search!
	res, err := s.e.Search(&eagle.SearchQuery{
		Query:    query,
		Sections: parsedSections,
		Draft:    false,
	}, page)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	s.serveJSON(w, http.StatusOK, res)
}
