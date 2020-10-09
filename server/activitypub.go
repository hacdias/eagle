package server

import (
	"net/http"
)

func (s *Server) activityPubInboxHandler(w http.ResponseWriter, r *http.Request) {
	// TODO(activitypub): implement inbox
	w.WriteHeader(http.StatusNotImplemented)
}
