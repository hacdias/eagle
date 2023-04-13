package server

import (
	"errors"
	"net/http"
	urlpkg "net/url"
	"os"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/eagle"
	"github.com/samber/lo"
)

const (
	newPath  = eaglePath + "/new"
	editPath = eaglePath + "/edit"
)

func (s *Server) newGet(w http.ResponseWriter, r *http.Request) {
	archetypeName := r.URL.Query().Get("archetype")
	if archetypeName == "" {
		archetypeName = "default"
	}

	archetype, ok := s.archetypes[archetypeName]
	if !ok {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("requested archetype does not exist"))
		return
	}

	e := archetype(s.c, r)

	// TODO: Override some properties according to query URL values.
	// if title := r.URL.Query().Get("title"); title != "" {
	// 	e.Title = title
	// }
	// if content := r.URL.Query().Get("content"); content != "" {
	// 	e.Content = content
	// }
	// if id := r.URL.Query().Get("id"); id != "" {
	// 	e.ID = id
	// }
	// for k, v := range r.URL.Query() {
	// 	if strings.HasPrefix(k, "properties.") {
	// 		e.Properties[strings.TrimPrefix(k, "properties.")] = v
	// 	}
	// }

	// Get stringified entry.
	str, err := e.String()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// Get all archetype names.
	archetypeNames := lo.Keys(s.archetypes)
	sort.Strings(archetypeNames)

	s.serveHTML(w, r, &RenderData{
		Title: "New",
		Data: map[string]interface{}{
			"ID":         e.ID,
			"Content":    str,
			"Archetypes": archetypeNames,
		},
	}, templateNew)
}

func (s *Server) newPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	now := time.Now().Local()
	id := r.FormValue("id")
	if id == "" {
		id = eagle.NewID("", now)
	}

	content := r.FormValue("content")
	if content == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content cannot be empty"))
		return
	}

	e, err := s.parser.Parse(id, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	if r.FormValue("published") != "" {
		e.Date = now
	}

	if location := r.FormValue("location"); location != "" {
		e.RawLocation = location
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

	go s.postSaveEntry(nil, e)
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
		http.Redirect(w, r, newPath+"?"+query.Encode(), http.StatusSeeOther)
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

	s.serveHTML(w, r, &RenderData{
		Title: "Edit",
		Data: map[string]interface{}{
			"Title":   ee.Title,
			"Content": str,
			"Entry":   ee,
		},
	}, templateEdit)
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

	e, err := s.parser.Parse(old.ID, content)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	lastmod := r.FormValue("lastmod") == "on"
	if lastmod {
		e.LastMod = time.Now().Local()
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

	go s.postSaveEntry(old, e)
	http.Redirect(w, r, e.ID, http.StatusSeeOther)
}
