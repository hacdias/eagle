package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"net/http"
)

func (s *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
	s.Debug("webhook: received request")

	signature := r.Header.Get("X-Hub-Signature")
	if len(signature) == 0 {
		s.Warn("webhook: request without signature")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil || len(payload) == 0 {
		s.Warn("webhook: invalid request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mac := hmac.New(sha1.New, []byte(s.c.Webhook.Secret))
	_, err = mac.Write(payload)
	if err != nil {
		s.Errorf("webook: could not write mac: %s", err)
		return
	}
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature[5:]), []byte(expectedMAC)) {
		s.Warn("webhook: forbidden request")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	go func() {
		s.Lock()
		defer s.Unlock()

		err := s.Store.Sync()
		if err != nil {
			s.Errorf("webhook: error git pull: %s", err)
			s.Notify.Error(err)
			return
		}

		err = s.Hugo.Build(false)
		if err != nil {
			s.Errorf("webhook: error hugo build: %s", err)
			s.Notify.Error(err)
			return
		}
	}()

	w.WriteHeader(http.StatusOK)
	s.Debug("webhook: request ok")
}
