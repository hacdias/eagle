package server

import (
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

var (
	commentsPath = "/comments/"
)

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
			Author: xray.Author{
				Name: name,
				URL:  website,
			},
			Content:   content,
			Published: time.Now(),
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
		s.n.Info(fmt.Sprintf("💬 #comment pending approval: %q", name))
	}
}

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
