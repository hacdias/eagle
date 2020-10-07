package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
)

func webmentionHandler(s *services.Services, c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wm := &services.WebmentionPayload{}
		err := json.NewDecoder(r.Body).Decode(&wm)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("webmention: error parsing: %s", err)
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
			log.Printf("webmention: error receiving: %s", err)
			return
		}

		w.WriteHeader(http.StatusOK)

		go func() {
			err := s.Hugo.Build(false)
			if err != nil {
				log.Printf("webmention: error hugo build: %s", err)
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
