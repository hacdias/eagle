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
		s.Errorf("webook: could not write mac: %w", err)
		return
	}
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature[5:]), []byte(expectedMAC)) {
		s.Warn("webhook: forbidden request")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	go s.syncStorage()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) syncStorage() {
	files, err := s.e.Sync()
	if err != nil {
		s.Errorf("sync storage: git pull: %w", err)
		s.e.NotifyError(err)
		return
	}

	if len(files) > 1 {
		// TODO: add one by one and detect removed files.
		// Maybe instead of wiping the complete index, simply
		// add everythign again - might be a problem when removing
		// posts.
		s.Infof("sync storage: files updated: %v", files)
		err = s.e.RebuildIndex()
		if err != nil {
			s.Errorf("sync storage: rebuild index: %w", err)
			s.e.NotifyError(err)
		}
	}

	err = s.e.Build(false)
	if err != nil {
		s.Errorf("sync storage: hugo build: %w", err)
		s.e.NotifyError(err)
		return
	}
}
