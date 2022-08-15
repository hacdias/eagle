package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/entry/mf2"
	"github.com/thoas/go-funk"
)

var (
	entryTemplates = map[string]func(r *http.Request) *entry.Entry{
		"default": func(r *http.Request) *entry.Entry {
			return &entry.Entry{
				Content: "Lorem ipsum...",
				Frontmatter: entry.Frontmatter{
					Published: time.Now().Local(),
				},
			}
		},
		"private": func(r *http.Request) *entry.Entry {
			return &entry.Entry{
				Content: "Lorem ipsum...",
				Frontmatter: entry.Frontmatter{
					Published: time.Now().Local(),
					Properties: map[string]interface{}{
						"visibility": "private",
						"audience":   "https://hacdias.com/",
						"category": []string{
							"example",
						},
					},
				},
			}
		},
		"now": func(r *http.Request) *entry.Entry {
			t := time.Now().Local()
			month := t.Format("January")

			return &entry.Entry{
				Content: "How was last month?",
				Frontmatter: entry.Frontmatter{
					Draft:     true,
					Title:     fmt.Sprintf("%s '%s", month, t.Format("06")),
					Published: t,
					Sections:  []string{"now"},
				},
			}
		},
		"article": func(r *http.Request) *entry.Entry {
			return &entry.Entry{
				Content: "Code is poetry...",
				Frontmatter: entry.Frontmatter{
					Draft:     true,
					Title:     "Article Title",
					Published: time.Now().Local(),
					Properties: map[string]interface{}{
						"category": []string{"example"},
					},
				},
			}
		},
		"book": func(r *http.Request) *entry.Entry {
			date := time.Now().Local()
			return &entry.Entry{
				ID: "/reads/isbn/ISBN",
				Frontmatter: entry.Frontmatter{
					Published:   date,
					Description: "NAME by AUTHOR (ISBN: ISBN)",
					Sections:    []string{"reads"},
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

	var ee *entry.Entry

	if fn, ok := entryTemplates[template]; ok {
		ee = fn(r)
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
		id = entry.NewID("", time.Now().Local())
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: &entry.Entry{},
		Data: map[string]interface{}{
			"ID":          id,
			"Content":     str,
			"Syndicators": s.GetSyndicators(),
			"Templates":   templates,
		},
		NoIndex: true,
	}, []string{eagle.TemplateNew})
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

	ee, err := s.Parser.FromRaw(id, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	ee.CreatedWith = s.Config.ID()

	if err := s.PreCreateEntry(ee); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	s.newEditHandler(w, r, ee)
}

func (s *Server) editGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "*")
	if id == "" {
		id = "/"
	}

	ee, err := s.GetEntry(id)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
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

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: &entry.Entry{
			ID: r.URL.Path,
		},
		Data: map[string]interface{}{
			"Title":       ee.Title,
			"Content":     str,
			"Syndicators": s.GetSyndicators(),
		},
		NoIndex: true,
	}, []string{eagle.TemplateEditor})
}

func (s *Server) editPost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "*")
	if id == "" {
		id = "/"
	}

	ee, err := s.GetEntry(id)
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

	ee, err = s.Parser.FromRaw(ee.ID, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	lastmod := r.FormValue("lastmod") == "on"
	if lastmod {
		ee.Updated = time.Now().Local()
	}

	s.newEditHandler(w, r, ee)
}

func (s *Server) newEditHandler(w http.ResponseWriter, r *http.Request, ee *entry.Entry) {
	syndications := r.Form["syndication"]

	if len(syndications) > 0 && ee.Draft {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("cannot syndicate draft entry"))
		return
	}

	if len(syndications) > 0 && ee.Visibility() == entry.VisibilityPrivate {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("cannot syndicate private entry"))
		return
	}

	if len(syndications) > 0 && ee.Deleted {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("cannot syndicate deleted entry"))
		return
	}

	err := s.SaveEntry(ee)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.PostSaveEntry(ee, syndications)
	http.Redirect(w, r, ee.ID, http.StatusSeeOther)
}

func (s *Server) entryGet(w http.ResponseWriter, r *http.Request) {
	ee, err := s.GetEntry(r.URL.Path)
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

	if ee.Visibility() == entry.VisibilityPrivate && !admin {
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

func (s *Server) serveEntry(w http.ResponseWriter, r *http.Request, ee *entry.Entry) {
	postType := ee.Helper().PostType()
	s.serveHTML(w, r, &eagle.RenderData{
		Entry:   ee,
		NoIndex: ee.NoIndex || ee.Visibility() != entry.VisibilityPublic || (postType != mf2.TypeNote && postType != mf2.TypeArticle),
	}, eagle.EntryTemplates(ee))
}
