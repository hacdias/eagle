package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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

	go s.syncStorage()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) syncStorage() {
	files, err := s.Sync()
	if err != nil {
		s.NotifyError(fmt.Errorf("sync storage: %w", err))
		return
	}

	if len(files) > 1 {
		// TODO: add one by one and detect removed files.
		// Maybe instead of wiping the complete index, simply
		// add everythign again - might be a problem when removing
		// posts.
		s.log.Infof("sync storage: files updated: %v", files)
		err = s.RebuildIndex()
		if err != nil {
			s.NotifyError(fmt.Errorf("sync storage: rebuild index: %w", err))
		}
	}

	err = s.Build(false)
	if err != nil {
		s.NotifyError(fmt.Errorf("sync storage: hugo build: %w", err))
		return
	}
}
