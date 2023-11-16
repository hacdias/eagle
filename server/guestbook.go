package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"go.hacdias.com/eagle/core"
)

var (
	guestbookPath     = "/guestbook/"
	guestbookFilename = filepath.Join(core.DataDirectory, "guestbook.json")
)

func (s *Server) guestbookPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("parse guestbook post request: %w", err))
		return
	}

	// Bot control honeypot field. If it's non-empty, just fake it was successful.
	if r.Form.Get("url") != "" {
		http.Redirect(w, r, r.URL.Path+"?youre=awesome", http.StatusFound)
		return
	}

	name := r.Form.Get("name")
	website := r.Form.Get("website")
	content := r.Form.Get("content")

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

	s.log.Infow("received guestbook entry", "name", name, "website", website, "content", content)

	err := s.badger.AddGuestbookEntry(r.Context(), &core.GuestbookEntry{
		Name:    name,
		Website: website,
		Content: content,
		Date:    time.Now(),
	})
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.n.Info(fmt.Sprintf("💬 #guestbook entry pending approval: %q", name))
	http.Redirect(w, r, r.URL.Path+"?youre=awesome", http.StatusFound)
}
