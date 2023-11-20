package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/xray"
)

const (
	webmentionPath = "/webmention"
	commentsPath   = "/comments"
)

var commentTemplate = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
	<head>
		<title>Comment Registered</title>
		<meta http-equiv="refresh" content="5; url={{ . }}">
	</head>
	<body>
		<p>
			Your comment has been registered and is pending moderation. You will be <a href="{{ . }}">redirected soon</a>.
	</body>
</html>`))

func (s *Server) commentsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("parse comment post request: %w", err))
		return
	}

	// Bot control honeypot field. If it's non-empty, just fake it was successful.
	if r.Form.Get("url") != "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("gotcha, bot"))
		return
	}

	name := r.Form.Get("name")
	website := r.Form.Get("website")
	content := r.Form.Get("content")
	target := r.Form.Get("target")

	if target == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("target is missing"))
		return
	}

	e, err := s.core.GetEntry(target)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("invalid target entry: %w", err))
		return
	}

	// Sanitize things just in case, specially the content.
	sanitize := bluemonday.StrictPolicy()
	name = sanitize.Sanitize(name)
	website = sanitize.Sanitize(website)
	content = sanitize.Sanitize(content)

	if len(content) == 0 || len(content) > 500 || len(name) > 100 || len(website) > 100 {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content, name, or website outside of limits"))
		return
	}

	if _, err := url.Parse(website); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("url is invalid: %w", err))
		return
	}

	s.log.Infow("received comment entry", "name", name, "website", website, "content", content)

	err = s.badger.AddMention(r.Context(), &core.Mention{
		Post: xray.Post{
			Author:    name,
			AuthorURL: website,
			Content:   content,
			Date:      time.Now(),
		},
		EntryID: e.ID,
	})
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	err = commentTemplate.Execute(w, e.Permalink)
	if err != nil {
		s.n.Error(fmt.Errorf("rendering comment redirect for %s: %w", r.URL.Path, err))
	} else {
		s.n.Info(fmt.Sprintf("💬 #mention pending approval for %q", e.Permalink))
	}
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

	err = s.badger.AddMention(context.Background(), mention)
	if err != nil {
		err = fmt.Errorf("could not add webmention for %s: %w", payload.Target, err)
		s.log.Errorf("webmention", err)
		s.n.Error(err)
	} else {
		s.n.Info(fmt.Sprintf("💬 #mention pending approval for %q: %q", e.Permalink, payload.Source))
	}
}
