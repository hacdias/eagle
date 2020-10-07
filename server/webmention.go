package server

import (
	"encoding/json"
	"net/http"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
)

func webmentionHanndler(s *services.Services, c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wm := &services.WebmentionPayload{}
		err := json.NewDecoder(r.Body).Decode(&wm)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if wm.Secret != c.WebmentionIO.Secret {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		wm.Secret = ""
		err = s.Webmentions.Receive(wm)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			// TODO: log
			return
		}

		w.WriteHeader(http.StatusOK)

		go func() {
			err := s.Hugo.Build(false)
			if err != nil {
				s.Notify.Error(err)
			} else {
				if wm.Deleted {
					s.Notify.Info("💬 Deleted webmention at " + wm.Target)
				} else {
					s.Notify.Info("💬 Received webmention at " + wm.Target)
				}
			}
		}()
	}
}
