package server

import (
	"encoding/json"
	"net/http"

	"github.com/hacdias/eagle/eagle"
)

func (s *Server) webmentionHandler(w http.ResponseWriter, r *http.Request) {
	s.Debug("webmention: received request")
	wm := &eagle.WebmentionPayload{}
	err := json.NewDecoder(r.Body).Decode(&wm)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.Errorf("webmention: error decoding: %s", err)
		return
	}

	if wm.Secret != s.c.WebmentionIO.Secret {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	s.Lock()
	defer s.Unlock()

	wm.Secret = ""
	err = s.ReceiveWebmentions(wm)
	if err != nil {
		if err == eagle.ErrDuplicatedWebmention {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		s.Errorf("webmention: error parsing: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	s.Debug("webmention: request ok")

	go func() {
		s.Lock()
		defer s.Unlock()

		err := s.Build(false)
		if err != nil {
			s.Errorf("webmention: error hugo build: %s", err)
			s.NotifyError(err)
		} else {
			if wm.Deleted {
				s.Notify("💬 Deleted webmention at " + wm.Target)
			} else {
				s.Notify("💬 Received webmention at " + wm.Target)
			}
		}
	}()
}
