package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/v2/eagle"
)

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	query, page, err := getSearchQuery(r)
	if err != nil {
		s.serveError(w, http.StatusBadRequest, err)
		return
	}

	// Do not accept empty searches!
	if query.Query == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Do not get drafts, nor deleted posts!
	f := false
	query.Draft = &f
	query.Deleted = &f

	// Search!
	res, err := s.Search(query, page)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	s.serveJSON(w, http.StatusOK, res)
}

func getSearchQuery(r *http.Request) (*eagle.SearchQuery, int, error) {
	q := r.URL.Query().Get("q")

	var err error
	p := 0
	if page := r.URL.Query().Get("p"); page != "" {
		p, err = strconv.Atoi(page)
		if err != nil {
			return nil, -1, err
		}
	}

	sectionsQuery := strings.TrimSpace(r.URL.Query().Get("s"))
	sectionsList := strings.Split(sectionsQuery, ",")
	sections := []string{}

	for _, s := range sectionsList {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		sections = append(sections, s)
	}

	return &eagle.SearchQuery{
		Query:    q,
		Sections: sections,
	}, p, nil
}
