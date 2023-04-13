package server

import (
	"fmt"
	"net/http"
)

const (
	deletedPath  = eaglePath + "/deleted"
	draftsPath   = eaglePath + "/drafts"
	unlistedPath = eaglePath + "/unlisted"
)

func (s *Server) draftsGet(w http.ResponseWriter, r *http.Request) {
	entries, err := s.i.GetDrafts(nil)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO
	fmt.Println(entries)
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) unlistedGet(w http.ResponseWriter, r *http.Request) {
	entries, err := s.i.GetUnlisted(nil)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO
	fmt.Println(entries)
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) deletedGet(w http.ResponseWriter, r *http.Request) {
	entries, err := s.i.GetDeleted(nil)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO
	fmt.Println(entries)
	w.WriteHeader(http.StatusNotImplemented)
}
