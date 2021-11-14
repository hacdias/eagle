package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"net/http"
)

func (s *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Hub-Signature")
	if len(signature) == 0 {
		s.log.Warn("webhook: request without signature")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil || len(payload) == 0 {
		s.log.Warn("webhook: invalid request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mac := hmac.New(sha1.New, []byte(s.Config.WebhookSecret))
	_, err = mac.Write(payload)
	if err != nil {
		s.log.Error("webook: could not write mac", err)
		return
	}
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature[5:]), []byte(expectedMAC)) {
		s.log.Warn("webhook: forbidden request")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	go s.SyncStorage()
	w.WriteHeader(http.StatusOK)
}
