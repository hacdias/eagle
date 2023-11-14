package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) serveErrorHTML(w http.ResponseWriter, r *http.Request, code int, reqErr error) {
	// TODO
	w.WriteHeader(code)
	if reqErr != nil {
		w.Write([]byte(reqErr.Error()))
	}
}

func (s *Server) serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.log.Error("serving json", err)
	}
}

func (s *Server) serveErrorJSON(w http.ResponseWriter, code int, err, errDescription string) {
	s.serveJSON(w, code, map[string]string{
		"error":             err,
		"error_description": errDescription,
	})
}
