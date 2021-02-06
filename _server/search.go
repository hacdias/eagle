package server

import (
	"net/http"
	"strconv"
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

	filter := r.URL.Query().Get("f")

	res, err := s.MeiliSearch.Search(query, filter, page)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	s.serveJSON(w, http.StatusOK, res)
}
