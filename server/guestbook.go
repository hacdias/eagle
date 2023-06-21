package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"time"

	"github.com/hacdias/eagle/core"
	"github.com/microcosm-cc/bluemonday"
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

func (s *Server) dashboardPostGuestbook(w http.ResponseWriter, r *http.Request) {
	action := r.Form.Get("guestbook-action")
	id := r.Form.Get("guestbook-id")

	switch action {
	case "approve":
		e, err := s.badger.GetGuestbookEntry(r.Context(), id)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		entries := core.GuestbookEntries{}
		if err := s.fs.ReadJSON(guestbookFilename, &entries); err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, fmt.Errorf("error reading guestbook file: %w", err))
			return
		}
		entries = append(entries, e)
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Date.After(entries[j].Date)
		})
		message := "guestbook: new entry"
		if e.Name != "" {
			message += " from " + e.Name
		}
		if err := s.fs.WriteJSON(guestbookFilename, entries, message); err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, fmt.Errorf("error writing guestbook file: %w", err))
			return
		}

		go func() {
			_ = s.hugo.Build(false)
		}()

		fallthrough
	case "delete":
		err := s.badger.DeleteGuestbookEntry(r.Context(), id)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}
	default:
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("invalid action: %s", action))
		return
	}

	http.Redirect(w, r, r.URL.Path, http.StatusFound)
}
