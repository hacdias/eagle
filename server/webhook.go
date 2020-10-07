package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"net/http"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
)

func webhookHandler(s *services.Services, c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		signature := r.Header.Get("X-Hub-Signature")
		if len(signature) == 0 {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		payload, err := ioutil.ReadAll(r.Body)
		if err != nil || len(payload) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mac := hmac.New(sha1.New, []byte(c.Webhook.Secret))
		_, _ = mac.Write(payload)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature[5:]), []byte(expectedMAC)) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		go func() {
			err := s.Git.Pull()
			if err != nil {
				s.Notify.Error(err)
				return
			}

			err = s.Hugo.Build(false)
			if err != nil {
				s.Notify.Error(err)
				return
			}
		}()

		w.WriteHeader(http.StatusOK)
	}
}
