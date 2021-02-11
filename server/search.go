package server

import (
	"net/http"
	"strconv"
	"strings"
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
	sectionsCond := []string{}

	for _, s := range sectionsList {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		sectionsCond = append(sectionsCond, "section=\""+s+"\"")
	}

	filter := ""
	if len(sectionsCond) > 0 {
		filter = "(" + strings.Join(sectionsCond, " OR ") + ")"
	}

	// Search!
	// TODO: maybe make the search api receive the sections instead of the raw filter.
	// Or some other {} struct.
	res, err := s.e.Search(query, filter, page)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	s.serveJSON(w, http.StatusOK, res)
}
