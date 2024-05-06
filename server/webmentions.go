package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/xray"
)

const (
	webmentionPath = "/webmention"
	commentsPath   = "/comments"
)

func (s *Server) commentsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("parse form failed: %w", err))
		return
	}

	// Anti-spam prevention with user-defined captcha value.
	if s.c.Comments.Captcha != "" && s.c.Comments.Captcha != strings.ToLower(r.Form.Get("captcha")) {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("anti-spam verification failed"))
		return
	}

	name := r.Form.Get("name")
	website := r.Form.Get("website")
	content := r.Form.Get("content")
	target := r.Form.Get("target")

	if target == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("target entry is missing"))
		return
	}

	e, err := s.core.GetEntry(target)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("target entry is invalid: %w", err))
		return
	}

	// Sanitize things just in case, specially the content.
	sanitize := bluemonday.StrictPolicy()
	name = sanitize.Sanitize(name)
	website = sanitize.Sanitize(website)
	content = sanitize.Sanitize(content)

	if len(name) == 0 || len(content) == 0 {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("name and content are required"))
		return
	}

	if len(content) > 1000 || len(name) > 200 || len(website) > 200 {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content, name, or website outside of limits"))
		return
	}

	if _, err := url.Parse(website); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("website url is invalid: %w", err))
		return
	}

	s.log.Infow("received comment entry", "name", name, "website", website, "content", content)

	err = s.bolt.AddMention(r.Context(), &core.Mention{
		Post: xray.Post{
			Author:    name,
			AuthorURL: website,
			Content:   content,
			Date:      time.Now(),
		},
		EntryID: e.ID,
	})
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.n.Info(fmt.Sprintf("💬 #mention pending approval for %q", e.Permalink))
	http.Redirect(w, r, s.c.Comments.Redirect, http.StatusSeeOther)
}

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

	err = s.bolt.AddMention(context.Background(), mention)
	if err != nil {
		err = fmt.Errorf("could not add webmention for %s: %w", payload.Target, err)
		s.log.Errorf("webmention", err)
		s.n.Error(err)
	} else {
		s.n.Info(fmt.Sprintf("💬 #mention pending approval for %q: %q", e.Permalink, payload.Source))
	}
}
