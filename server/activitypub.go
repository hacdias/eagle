package server

import (
	"encoding/json"
	"net/http"

	"github.com/hacdias/eagle/eagle"
)

func (s *Server) activityPubPostInboxHandler(w http.ResponseWriter, r *http.Request) {
	var activity map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&activity)
	if err != nil {
		s.serveError(w, http.StatusBadRequest, err)
		return
	}

	s.Lock()
	defer s.Unlock()

	var msg string

	// TODO: check if request is signed by the actual user
	// to prevent misuse of this endpoint.

	switch activity["type"] {
	case "Follow":
		msg, err = s.ActivityPub.Follow(activity)
	case "Create":
		err = s.ActivityPub.Create(activity)
	case "Like":
		msg, err = s.ActivityPub.Like(activity)
	case "Delete":
		msg, err = s.ActivityPub.Delete(activity)
	case "Undo":
		msg, err = s.ActivityPub.Undo(activity)
	default:
		err = eagle.ErrNotHandled
	}

	if err == eagle.ErrNoChanges {
		w.WriteHeader(http.StatusCreated)
		return
	}

	if err == eagle.ErrNotHandled {
		msg = "Received unhandled Activity"
		err = s.ActivityPub.Log(activity)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Errorf("activity inbox: %s", err)
		s.NotifyError(err)
		return
	}

	if msg != "" {
		s.Notify(msg)
	}

	err = s.Persist("activitypub")
	if err != nil {
		s.Errorf("activitypub: error git commit: %s", err)
		s.NotifyError(err)
	}

	w.WriteHeader(http.StatusCreated)
}
