package server

import (
	"bytes"
	"errors"
	"net/http"
	urlpkg "net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/renderer"
	"github.com/samber/lo"
)

func (s *Server) newGet(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")
	if template == "" {
		template = "default"
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		id = eagle.NewID("", time.Now().Local())
	}

	var str string

	at, ok := s.archetypes[template]
	if !ok {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("requested template does not exist"))
		return
	}

	var buf bytes.Buffer

	err := at.Execute(&buf, map[string]interface{}{
		"Now":   time.Now(),
		"Query": r.URL.Query(),
	})
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	e, err := s.parser.FromRaw(id, buf.String())
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// Override some properties according to query URL values.
	// TODO: perhaps there's some way of simplifying this for all fields that can
	// be strings in the FrontMatter.
	for k, v := range r.URL.Query() {
		if strings.HasPrefix(k, "properties.") {
			e.Properties[strings.TrimPrefix(k, "properties.")] = v
		}
	}
	if title := r.URL.Query().Get("title"); title != "" {
		e.Title = title
	}
	if content := r.URL.Query().Get("content"); content != "" {
		e.Content = content
	}

	str, err = e.String()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	templates := lo.Keys(s.archetypes)
	sort.Strings(templates)

	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &eagle.Entry{
			FrontMatter: eagle.FrontMatter{
				Title: "New",
			},
		},
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

	e, err := s.parser.FromRaw(id, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	e.CreatedWith = s.c.ID()

	if location := r.FormValue("location"); location != "" {
		e.Properties["location"] = location
	}

	if err := s.preSaveEntry(nil, e); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	err = s.fs.SaveEntry(e)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.postSaveEntry(nil, e, r.Form["syndication"])
	http.Redirect(w, r, e.ID, http.StatusSeeOther)
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
		Entry: &eagle.Entry{
			FrontMatter: eagle.FrontMatter{
				Title: "Edit",
			},
		},
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

	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	rename := r.Form.Get("rename")
	if rename != "" {
		ne, err := s.fs.RenameEntry(id, rename)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		http.Redirect(w, r, ne.ID, http.StatusSeeOther)
		return
	}

	old, err := s.fs.GetEntry(id)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	content := r.Form.Get("content")
	if content == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content cannot be empty"))
		return
	}

	e, err := s.parser.FromRaw(old.ID, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	lastmod := r.FormValue("lastmod") == "on"
	if lastmod {
		e.Updated = time.Now().Local()
	}

	if err := s.preSaveEntry(old, e); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	err = s.fs.SaveEntry(e)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.postSaveEntry(old, e, r.Form["syndication"])
	http.Redirect(w, r, e.ID, http.StatusSeeOther)
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

		if hasAudience && !lo.Contains(ee.Audience(), user) {
			s.serveErrorHTML(w, r, http.StatusForbidden, nil)
			return
		}
	}

	if s.ap != nil && isActivityPub(r) {
		s.serveActivity(w, http.StatusAccepted, s.ap.GetEntryAsActivity(ee))
		return
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
