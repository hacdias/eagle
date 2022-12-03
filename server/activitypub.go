package server

import (
	"errors"
	"net/http"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/renderer"
)

var (
	activityPubInboxRoute        = "/activitypub/inbox"
	activityPubOutboxRoute       = "/activitypub/outbox"
	activityPubFollowersRoute    = "/activitypub/followers"
	activityPubRemoteFollowRoute = "/activitypub/remote-follow"
)

func (s *Server) serveActivityError(w http.ResponseWriter, statusCode int, err error) {
	if statusCode >= 500 {
		s.serveErrorJSON(w, statusCode, "error", "internals server error")
		s.log.Errorw("activity", "status", statusCode, "err", err)
	} else {
		s.serveErrorJSON(w, statusCode, "error", err.Error())
	}
}

func (s *Server) activityPubInboxPost(w http.ResponseWriter, r *http.Request) {
	statusCode, err := s.ap.InboxHandler(r)
	if err != nil {
		s.serveActivityError(w, statusCode, err)
	} else if statusCode != 0 {
		w.WriteHeader(statusCode)
	}
}

func (s *Server) activityPubFollowersGet(w http.ResponseWriter, r *http.Request) {
	if isActivityPub(r) {
		statusCode, err := s.ap.FollowersHandler(w, r)
		if err != nil {
			s.serveActivityError(w, statusCode, err)
		} else if statusCode != 0 {
			w.WriteHeader(statusCode)
		}
		return
	}

	followers, err := s.ap.Store.GetFollowers()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &eagle.Entry{
			FrontMatter: eagle.FrontMatter{
				Title: "Followers",
			},
		},
		Data: map[string]interface{}{
			"Followers": followers,
		},
		NoIndex: true,
	}, []string{renderer.TemplateActivityPubFollowers})
}

func (s *Server) activityPubRemoteFollowPost(w http.ResponseWriter, r *http.Request) {
	statusCode, err := s.ap.RemoteFollowHandler(w, r)
	if err != nil {
		s.serveErrorHTML(w, r, statusCode, err)
	} else if statusCode != 0 {
		w.WriteHeader(statusCode)
	}
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
