package server

import (
	"encoding/json"
	"net/http"

	"github.com/hacdias/eagle/v4/eagle"
)

func (s *Server) webmentionPost(w http.ResponseWriter, r *http.Request) {
	wm := &eagle.WebmentionPayload{}
	err := json.NewDecoder(r.Body).Decode(&wm)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.log.Warnf("error when decoding webmention: %s", err.Error())
		return
	}

	if wm.Secret != s.Config.Webmentions.Secret {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	wm.Secret = ""
	err = s.ReceiveWebmentions(wm)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.log.Error("could not parse webmention", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
