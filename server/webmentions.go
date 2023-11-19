package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/xray"
)

const (
	webmentionPath = "/webmention"
)

type webmentionPayload struct {
	Source  string                 `json:"source"`
	Secret  string                 `json:"secret"`
	Deleted bool                   `json:"deleted"`
	Target  string                 `json:"target"`
	Post    map[string]interface{} `json:"post"`
}

func (s *Server) webmentionPost(w http.ResponseWriter, r *http.Request) {
	payload := &webmentionPayload{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.log.Warnf("error when decoding webmention: %s", err.Error())
		return
	}

	if payload.Secret != s.c.Webmentions.Secret {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	payload.Secret = ""
	go s.handleWebmention(payload)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleWebmention(payload *webmentionPayload) {
	s.log.Infow("received webmention", "webmention", payload)
	e, err := s.core.GetEntryFromPermalink(payload.Target)
	if err != nil {
		err = fmt.Errorf("could not get entry for permalink %s: %w", payload.Target, err)
		s.log.Errorf("webmention", err)
		s.n.Error(err)
		return
	}

	if payload.Deleted {
		err = s.core.DeleteWebmention(e.ID, payload.Source)
		if err != nil {
			err = fmt.Errorf("could not delete webmention for %s: %w", payload.Target, err)
			s.log.Errorf("webmention", err)
			s.n.Error(err)
		} else {
			s.n.Info(fmt.Sprintf("💬 #mention deleted for %q: %q", e.Permalink, payload.Source))
		}
		return
	}

	mention := &core.Mention{
		Post:    *xray.Parse(payload.Post),
		EntryID: e.ID,
	}

	if payload.Source != mention.Post.URL {
		mention.Source = payload.Source
	}

	err = s.badger.AddMention(context.Background(), mention)
	if err != nil {
		err = fmt.Errorf("could not add webmention for %s: %w", payload.Target, err)
		s.log.Errorf("webmention", err)
		s.n.Error(err)
	} else {
		s.n.Info(fmt.Sprintf("💬 #mention added or updated for %q: %q", e.Permalink, payload.Source))
	}
}
