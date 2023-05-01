package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/hacdias/eagle/core"
	"github.com/microcosm-cc/bluemonday"
)

type guestbookStorage interface {
	AddGuestbookEntry(ctx context.Context, entry *core.GuestbookEntry) error
	GetGuestbookEntry(ctx context.Context, id int) (core.GuestbookEntry, error)
	GetGuestbookEntries(ctx context.Context) (core.GuestbookEntries, error)
	DeleteGuestbookEntry(ctx context.Context, id int) error
}

var (
	guestbookPath      = "/guestbook/"
	guestbookAdminPath = guestbookPath + "admin/"
	guestbookFilename  = filepath.Join(core.DataDirectory, "guestbook.json")
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

	err := s.guestbook.AddGuestbookEntry(r.Context(), &core.GuestbookEntry{
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

func (s *Server) guestbookAdminGet(w http.ResponseWriter, r *http.Request) {
	doc, err := s.getTemplateDocument(r.URL.Path)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	ee, err := s.guestbook.GetGuestbookEntries(r.Context())
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, fmt.Errorf("error getting guestbook entries: %w", err))
		return
	}

	entriesNode := doc.Find("eagle-guestbook-entries")
	entryTemplate := doc.Find("eagle-guestbook-entry")
	entriesNode.Empty()

	for _, e := range ee {
		node := entryTemplate.Clone()
		node.Find("eagle-guestbook-name").ReplaceWithHtml(e.Name)
		node.Find("eagle-guestbook-website").ReplaceWithHtml(e.Website)
		node.Find("eagle-guestbook-date").ReplaceWithHtml(e.Date.String())
		node.Find("eagle-guestbook-content").ReplaceWithHtml(e.Content)
		node.Find("input[name='id']").SetAttr("value", strconv.Itoa(e.ID))
		entriesNode.AppendSelection(node.Children())
	}

	entriesNode.ReplaceWithSelection(entriesNode.Children())
	s.serveDocument(w, r, doc, http.StatusOK)
}

func (s *Server) guestbookAdminPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("error parsing guestbook admin post: %w", err))
		return
	}

	action := r.Form.Get("action")
	id, err := strconv.Atoi(r.Form.Get("id"))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	switch action {
	case "approve":
		e, err := s.guestbook.GetGuestbookEntry(r.Context(), id)
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
		if err := s.fs.WriteJSON(guestbookFilename, entries); err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, fmt.Errorf("error writing guestbook file: %w", err))
			return
		}

		go func() {
			_ = s.hugo.Build(false)
		}()

		fallthrough
	case "delete":
		err = s.guestbook.DeleteGuestbookEntry(r.Context(), id)
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
