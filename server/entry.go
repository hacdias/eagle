package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/entry/mf2"
	"github.com/thoas/go-funk"
)

var (
	entryTemplates = map[string]func() *entry.Entry{
		"default": func() *entry.Entry {
			return &entry.Entry{
				Content: "Lorem ipsum...",
				Frontmatter: entry.Frontmatter{
					Published: time.Now(),
				},
			}
		},
		"recently": func() *entry.Entry {
			t := time.Now()
			month := t.Format("January")

			return &entry.Entry{
				Content: "How was last month?",
				Frontmatter: entry.Frontmatter{
					Draft:     true,
					Title:     fmt.Sprintf("Recently in %s '%s", month, t.Format("06")),
					Published: t,
					Properties: map[string]interface{}{
						"categories": []string{"recently"},
					},
				},
			}
		},
		"article": func() *entry.Entry {
			return &entry.Entry{
				Content: "Code is poetry...",
				Frontmatter: entry.Frontmatter{
					Draft:     true,
					Title:     "Article Title",
					Published: time.Now(),
					Properties: map[string]interface{}{
						"categories": []string{"example"},
					},
				},
			}
		},
		"book": func() *entry.Entry {
			return &entry.Entry{
				ID: "/reads/isbn/ISBN",
				Frontmatter: entry.Frontmatter{
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
					},
				},
			}
		},
		"want-to-read": func() *entry.Entry {
			return &entry.Entry{
				Frontmatter: entry.Frontmatter{
					Published: time.Now(),
					Sections:  []string{"reads"},
					Properties: map[string]interface{}{
						"read-status": "to-read",
						"read-of": map[string]interface{}{
							"properties": map[string]interface{}{
								"author": "AUTHOR",
								"name":   "NAME",
							},
							"type": "h-cite",
						},
					},
				},
			}
		},
		"currently-reading": func() *entry.Entry {
			return &entry.Entry{
				Frontmatter: entry.Frontmatter{
					Published: time.Now(),
					Sections:  []string{"reads"},
					Properties: map[string]interface{}{
						"read-status": "reading",
						"page":        "PAGE",
						"read-of":     "/reads/isbn/ISBN",
					},
				},
			}
		},
		"finished-reading": func() *entry.Entry {
			return &entry.Entry{
				Frontmatter: entry.Frontmatter{
					Published: time.Now(),
					Sections:  []string{"reads"},
					Properties: map[string]interface{}{
						"read-status": "finished",
						"rating":      "RATING",
						"read-of":     "/reads/isbn/ISBN",
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
		ee = fn()
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
		id = entry.NewID("", time.Now())
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

	if err := s.newHandler(ee); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	s.newEditHandler(w, r, ee)
}

func (s *Server) newHandler(ee *entry.Entry) error {
	if ee.Description != "" {
		return nil
	}

	mm := ee.Helper()
	if mm.PostType() != mf2.TypeRead {
		return nil
	}

	status := mm.String("read-status")
	if status == "" {
		return nil
	}

	description := ""

	switch status {
	case "to-read":
		description = "Want to read"
	case "reading":
		description = "Currently reading"
	case "finished":
		description = "Finished reading"
	}

	sub := mm.Sub(mm.TypeProperty())
	if sub == nil {
		canonical := mm.String(mm.TypeProperty())
		e, err := s.GetEntry(canonical)
		if err != nil {
			return err
		}
		sub = e.Helper().Sub(mm.TypeProperty())
	}

	if sub == nil {
		return nil
	}

	name := sub.String("name")
	author := sub.String("author")
	uid := sub.String("uid")

	description += ": " + name + " by " + author

	if uid != "" {
		parts := strings.Split(uid, ":")
		if len(parts) == 2 {
			description += ", " + strings.ToUpper(parts[0]) + ": " + parts[1]
		}
	}

	ee.Description = description
	return nil
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
		Entry: &entry.Entry{},
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
		ee.Updated = time.Now()
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

func (s *Server) serveEntry(w http.ResponseWriter, r *http.Request, entry *entry.Entry) {
	s.serveHTML(w, r, &eagle.RenderData{
		Entry: entry,
	}, eagle.EntryTemplates(entry))
}
