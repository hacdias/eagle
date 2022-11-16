package server

import (
	"net/http"
)

func (s *Server) activityPubInboxPost(w http.ResponseWriter, r *http.Request) {
	statusCode, err := s.ap.HandleInbox(r)
	if err != nil {
		s.log.Errorw("activity", "status", statusCode, "err", err)
		s.serveErrorJSON(w, statusCode, "invalid_request", err.Error())
		return
	}

	w.WriteHeader(statusCode)
}

func (s *Server) activityPubOutboxGet(w http.ResponseWriter, r *http.Request) {
	// TODO: integrate this somehow with the activitypub package.
	countBySection, err := s.i.CountBySection()
	if err != nil {
		s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	total := 0
	for _, v := range countBySection {
		total += v
	}

	s.serveActivity(w, http.StatusOK, map[string]interface{}{
		"@context": []string{
			"https://www.w3.org/ns/activitystreams",
		},
		"id":         s.c.Server.AbsoluteURL("/activitypub/outbox"),
		"type":       "OrderedCollection",
		"totalItems": total,
	})
}
