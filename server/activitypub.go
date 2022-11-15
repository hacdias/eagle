package server

import (
	"net/http"
)

func (s *Server) activityPubInboxPost(w http.ResponseWriter, r *http.Request) {
	statusCode, err := s.ap.HandleInbox(r)
	if err != nil {
		s.serveErrorJSON(w, statusCode, "invalid_request", err.Error())
		return
	}

	w.WriteHeader(statusCode)
}
