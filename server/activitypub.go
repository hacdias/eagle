package server

import (
	"encoding/json"
	"net/http"

	"github.com/hacdias/eagle/services"
)

func (s *Server) activityPubPostInboxHandler(w http.ResponseWriter, r *http.Request) {
	var activity map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&activity)
	if err != nil {
		s.serveError(w, http.StatusBadRequest, err)
		return
	}

	switch activity["type"] {
	case "Follow":
		err = s.ActivityPub.Follow(activity)
	case "Create":
		err = s.ActivityPub.Create(activity)
	case "Like":
		err = s.ActivityPub.Like(activity)
	case "Delete":
		err = s.ActivityPub.Delete(activity)
	case "Undo":
		err = s.ActivityPub.Undo(activity)
	default:
		err = services.ErrNotHandled
	}

	if err == services.ErrNotHandled {
		// TODO: log it
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.Errorf("activity inbox: %s", err)
		s.Notify.Error(err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
