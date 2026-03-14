package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

const (
	webhookPath = "/webhook"
)

func (s *Server) webhookPost(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Hub-Signature-256")
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

	mac := hmac.New(sha256.New, []byte(s.c.WebhookSecret))
	_, err = mac.Write(payload)
	if err != nil {
		s.log.Errorw("webhook: could not write mac", "err", err)
		return
	}
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	const signaturePrefix = "sha256="
	if !strings.HasPrefix(signature, signaturePrefix) {
		s.log.Warn("webhook: invalid signature format")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if !hmac.Equal([]byte(signature[len(signaturePrefix):]), []byte(expectedMAC)) {
		s.log.Warn("webhook: forbidden request")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	go s.syncStorage()
	w.WriteHeader(http.StatusOK)
}
