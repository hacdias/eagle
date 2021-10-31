package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hacdias/eagle/eagle"
)

func (s *Server) webmentionHandler(w http.ResponseWriter, r *http.Request) {
	wm := &eagle.WebmentionPayload{}
	err := json.NewDecoder(r.Body).Decode(&wm)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.log.Warnf("error when decoding webmention: %s", err.Error())
		return
	}

	if wm.Secret != s.Config.WebmentionsSecret {
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

	go func() {
		err := s.Build(false)
		if err != nil {
			s.NotifyError(fmt.Errorf("webmention: error hugo build: %w", err))
		} else {
			if wm.Deleted {
				s.Notify("💬 Deleted webmention at " + wm.Target)
			} else {
				s.Notify("💬 Received webmention at " + wm.Target)
			}
		}
	}()
}
