package server

import (
	"encoding/json"
	"net/http"

	"github.com/hacdias/eagle/webmentions"
)

const (
	webmentionPath = "/webmention"
)

func (s *Server) webmentionPost(w http.ResponseWriter, r *http.Request) {
	wm := &webmentions.Payload{}
	err := json.NewDecoder(r.Body).Decode(&wm)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.log.Warnf("error when decoding webmention: %s", err.Error())
		return
	}

	if wm.Secret != s.c.Webmentions.Secret {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	wm.Secret = ""
	err = s.webmentions.ReceiveWebmentions(wm)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.log.Error("could not parse webmention", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
