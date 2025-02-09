package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
)

const (
	webhookPath = "/webhook"
)

func (s *Server) webhookPost(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Hub-Signature")
	if len(signature) == 0 {
		s.log.Warn("webhook: request without signature")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil || len(payload) == 0 {
		s.log.Warn("webhook: invalid request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mac := hmac.New(sha1.New, []byte(s.c.WebhookSecret))
	_, err = mac.Write(payload)
	if err != nil {
		s.log.Errorw("webhook: could not write mac", "err", err)
		return
	}
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature[5:]), []byte(expectedMAC)) {
		s.log.Warn("webhook: forbidden request")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	go s.syncStorage()
	w.WriteHeader(http.StatusOK)
}
