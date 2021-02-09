package server

import (
	"net/http"
	"time"
)

func (s *Server) editorGetHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	_ = r.URL.Query().Get("reply")
	_ = r.URL.Query().Get("template")

	entry, err := s.eagle.GetEntry(id)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	w.Write([]byte(entry.Content))
}

func (s *Server) editorDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	s.Lock()
	defer s.Unlock()

	entry, err := s.eagle.GetEntry(id)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	entry.Metadata.ExpiryDate = time.Now()
	err = s.eagle.SaveEntry(entry)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	s.eagle.Build(true)
	http.Redirect(w, r, id, http.StatusTemporaryRedirect)
}

func (s *Server) editorPostHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: update/new
}
