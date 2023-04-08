package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/microcosm-cc/bluemonday"
)

const (
	guestbookFilename = "/content/guestbook/.entries.json"
)

func (s *Server) guestbookPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	name := r.Form.Get("name")
	website := r.Form.Get("website")
	content := r.Form.Get("content")
	content = bluemonday.StrictPolicy().Sanitize(content)

	if content == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content must not be missing"))
		return
	}

	if _, err := url.Parse(website); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("url is invalid: %w", err))
		return
	}

	s.log.Infow("received guestbook entry", "name", name, "website", website, "content", content)

	entries := eagle.GuestbookEntries{}

	if err := s.fs.ReadJSON(guestbookFilename, &entries); err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	entries = append(entries, eagle.GuestbookEntry{
		Name:    name,
		Website: website,
		Content: content,
		Date:    time.Now(),
		Unseen:  true,
	})

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Date.After(entries[j].Date)
	})

	if err := s.fs.WriteJSON(guestbookFilename, entries); err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.n.Info(fmt.Sprintf("💬 #guestbook entry pending approval: %s.", content))
	http.Redirect(w, r, r.URL.Path+"?youre=awesome", http.StatusFound)
}
