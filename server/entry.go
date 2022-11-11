package server

import (
	"errors"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"os"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/renderer"
	"github.com/thoas/go-funk"
)

var (
	// TODO: make this archetypes instead of hard-coding them.
	entryTemplates = map[string]func(r *http.Request, s *Server) *eagle.Entry{
		"default": func(r *http.Request, s *Server) *eagle.Entry {
			return &eagle.Entry{
				Content: "What's on your mind?",
				FrontMatter: eagle.FrontMatter{
					Published: time.Now().Local(),
				},
			}
		},
		"private": func(r *http.Request, s *Server) *eagle.Entry {
			return &eagle.Entry{
				Content: "What's on your mind?",
				FrontMatter: eagle.FrontMatter{
					Published: time.Now().Local(),
					Properties: map[string]interface{}{
						"visibility": "private",
						"audience":   s.getUser(r),
					},
				},
			}
		},
		"now": func(r *http.Request, s *Server) *eagle.Entry {
			t := time.Now().Local()
			month := t.Format("January")

			return &eagle.Entry{
				Content: "How was last month?",
				FrontMatter: eagle.FrontMatter{
					Draft:     true,
					Title:     fmt.Sprintf("Recently in %s '%s", month, t.Format("06")),
					Published: t,
					Sections:  []string{"home", "now"},
				},
			}
		},
		"article": func(r *http.Request, s *Server) *eagle.Entry {
			return &eagle.Entry{
				Content: "Code is poetry...",
				FrontMatter: eagle.FrontMatter{
					Draft:     true,
					Title:     "Article Title",
					Published: time.Now().Local(),
					Properties: map[string]interface{}{
						"category": []string{"example"},
					},
				},
			}
		},
		"book": func(r *http.Request, s *Server) *eagle.Entry {
			date := time.Now().Local()
			return &eagle.Entry{
				ID: "/books/BOOK-NAME-SLUG",
				FrontMatter: eagle.FrontMatter{
					Published:   date,
					Description: "NAME by AUTHOR (ISBN: ISBN)",
					Sections:    []string{"books"},
					Properties: map[string]interface{}{
						"read-of": map[string]interface{}{
							"properties": map[string]interface{}{
								"author":    "AUTHOR",
								"name":      "NAME",
								"pages":     "PAGES",
								"publisher": "PUBLISHER",
								"uid":       "isbn:ISBN",
							},
							"type": "h-cite",
						},
						"read-status": []interface{}{
							map[string]interface{}{
								"status": "to-read",
								"date":   date,
							},
						},
					},
				},
			}
		},
	}
)

func (s *Server) newGet(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")
	if template == "" {
		template = "default"
	}

	var ee *eagle.Entry

	if fn, ok := entryTemplates[template]; ok {
		ee = fn(r, s)
	} else {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("requested template does not exist"))
		return
	}

	str, err := ee.String()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	templates := funk.Keys(entryTemplates).([]string)
	sort.Strings(templates)

	id := ee.ID
	if id == "" {
		if qid := r.URL.Query().Get("id"); qid != "" {
			id = qid
		} else {
			id = eagle.NewID("", time.Now().Local())
		}
	}

	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &eagle.Entry{},
		Data: map[string]interface{}{
			"ID":          id,
			"Content":     str,
			"Syndicators": s.syndicator.Config(),
			"Templates":   templates,
		},
		NoIndex: true,
	}, []string{renderer.TemplateNew})
}

func (s *Server) newPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	content := r.FormValue("content")
	id := r.FormValue("id")
	if content == "" || id == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content and slug cannot be empty"))
		return
	}

	ee, err := s.parser.FromRaw(id, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	ee.CreatedWith = s.c.ID()

	if err := s.preSaveEntry(ee, true); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	err = s.fs.SaveEntry(ee)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.postSaveEntry(ee, true, r.Form["syndication"])
	http.Redirect(w, r, ee.ID, http.StatusSeeOther)
}

func (s *Server) editGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "*")
	if id == "" {
		id = "/"
	}

	ee, err := s.fs.GetEntry(id)
	if os.IsNotExist(err) {
		query := urlpkg.Values{}
		query.Set("id", id)
		http.Redirect(w, r, "/new?"+query.Encode(), http.StatusSeeOther)
		return
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	str, err := ee.String()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &eagle.Entry{},
		Data: map[string]interface{}{
			"Title":       ee.Title,
			"Content":     str,
			"Entry":       ee,
			"Syndicators": s.syndicator.Config(),
		},
		NoIndex: true,
	}, []string{renderer.TemplateEditor})
}

func (s *Server) editPost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "*")
	if id == "" {
		id = "/"
	}

	ee, err := s.fs.GetEntry(id)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	err = r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	content := r.Form.Get("content")
	if content == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content cannot be empty"))
		return
	}

	ee, err = s.parser.FromRaw(ee.ID, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	lastmod := r.FormValue("lastmod") == "on"
	if lastmod {
		ee.Updated = time.Now().Local()
	}

	if err := s.preSaveEntry(ee, false); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	err = s.fs.SaveEntry(ee)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.postSaveEntry(ee, false, r.Form["syndication"])
	http.Redirect(w, r, ee.ID, http.StatusSeeOther)
}

func (s *Server) entryGet(w http.ResponseWriter, r *http.Request) {
	ee, err := s.fs.GetEntry(r.URL.Path)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	admin := s.isAdmin(r)
	if ee.Deleted && !admin {
		s.serveErrorHTML(w, r, http.StatusGone, nil)
		return
	}

	if ee.Draft && !admin {
		s.serveErrorHTML(w, r, http.StatusForbidden, nil)
		return
	}

	if ee.Visibility() == eagle.VisibilityPrivate && !admin {
		user := s.getUser(r)
		hasUser := user != ""
		hasAudience := len(ee.Audience()) != 0

		if !hasUser {
			s.serveErrorHTML(w, r, http.StatusForbidden, nil)
			return
		}

		if hasAudience && !funk.ContainsString(ee.Audience(), user) {
			s.serveErrorHTML(w, r, http.StatusForbidden, nil)
			return
		}
	}

	s.serveEntry(w, r, ee)
}

func (s *Server) serveEntry(w http.ResponseWriter, r *http.Request, ee *eagle.Entry) {
	postType := ee.Helper().PostType()
	s.serveHTML(w, r, &renderer.RenderData{
		Entry:   ee,
		NoIndex: ee.NoIndex || ee.Visibility() != eagle.VisibilityPublic || (postType != mf2.TypeNote && postType != mf2.TypeArticle),
	}, renderer.EntryTemplates(ee))
}

func (s *Server) renameGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "*")
	if id == "" {
		id = "/"
	}

	ee, err := s.fs.GetEntry(id)
	if os.IsNotExist(err) {
		query := urlpkg.Values{}
		query.Set("id", id)
		http.Redirect(w, r, "/new?"+query.Encode(), http.StatusSeeOther)
		return
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &eagle.Entry{},
		Data: map[string]interface{}{
			"Title": ee.Title,
			"Entry": ee,
		},
		NoIndex: true,
	}, []string{renderer.TemplateRename})
}

func (s *Server) renamePost(w http.ResponseWriter, r *http.Request) {
	oldID := chi.URLParam(r, "*")
	if oldID == "" {
		oldID = "/"
	}

	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	newID := r.Form.Get("id")
	if newID == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("id cannot be empty"))
		return
	}

	e, err := s.fs.RenameEntry(oldID, newID)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, e.ID, http.StatusSeeOther)
}

func (s *Server) mentionToggleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "*")
	if id == "" {
		id = "/"
	}

	wm := r.URL.Query().Get("wm")
	if wm == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("entry id or webmention url missing"))
		return
	}

	ee, err := s.fs.GetEntry(id)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	err = s.fs.UpdateSidecar(ee, func(s *eagle.Sidecar) (*eagle.Sidecar, error) {
		for i := range s.Replies {
			if s.Replies[i].URL == wm {
				s.Replies[i].Hidden = !s.Replies[i].Hidden
				return s, nil
			}
		}

		for i := range s.Interactions {
			if s.Interactions[i].URL == wm {
				s.Interactions[i].Hidden = !s.Interactions[i].Hidden
				return s, nil
			}
		}

		return nil, errors.New("webmention not found")
	})

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, ee.ID, http.StatusSeeOther)
}
