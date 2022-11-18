package server

import (
	"errors"
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
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	id := r.Form.Get("id")
	if id == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("id is missing"))
		return
	}

	e, err := s.fs.GetEntry(id)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	action := r.Form.Get("action")

	switch action {
	case "create":
		err = s.ap.SendCreate(e)
	case "update":
		err = s.ap.SendUpdate(e)
	case "announce":
		err = s.ap.SendAnnounce(e)
	case "delete":
		err = s.ap.SendDelete(e.ID)
	default:
		err = errors.New("invalid action")
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	http.Redirect(w, r, e.Permalink, http.StatusSeeOther)
}
