package server

import (
	"encoding/json"
	"errors"
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

	// TODO: check if request is signed by the actual user
	// to prevent misuse of this endpoint.

	switch activity["type"] {
	case "Follow":
		err = s.e.ActivityPub.Follow(activity)
	case "Create":
		err = s.e.ActivityPub.Create(activity)
	case "Like":
		err = s.e.ActivityPub.Like(activity)
	case "Delete":
		err = s.e.ActivityPub.Delete(activity)
	case "Undo":
		err = s.e.ActivityPub.Undo(activity)
	default:
		err = eagle.ErrNotHandled
	}

	if err == nil {
		w.WriteHeader(http.StatusCreated)
		return
	}

	if errors.Is(err, eagle.ErrNotHandled) {
		s.e.Notify("Received unhandled ActivityPub object")
		err = s.e.ActivityPub.Log(activity)
	}

	w.WriteHeader(http.StatusInternalServerError)
	s.Errorf("activity handler: %w", err)
	s.e.NotifyError(err)
}
