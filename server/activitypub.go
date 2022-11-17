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
	count, err := s.i.Count()
	if err != nil {
		s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	s.serveActivity(w, http.StatusOK, map[string]interface{}{
		"@context": []string{
			"https://www.w3.org/ns/activitystreams",
		},
		"id":         s.c.Server.AbsoluteURL("/activitypub/outbox"),
		"type":       "OrderedCollection",
		"totalItems": count,
	})
}

func (s *Server) activityPubHookPost(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "id is missing")
		return
	}

	e, err := s.fs.GetEntry(id)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	action := r.URL.Query().Get("action")

	switch action {
	case "create":
		s.ap.SendCreate(e)
	case "update":
		s.ap.SendUpdate(e)
	case "announce":
		s.ap.SendAnnounce(e)
	case "delete":
		s.ap.SendDelete(e.ID)
	default:
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "invalid action")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
