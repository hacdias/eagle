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
	"github.com/samber/lo"
)

const (
	newPath  = "/new/"
	editPath = "/edit/"
)

func (s *Server) newGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	archetypeName := query.Get("archetype")
	if archetypeName == "" {
		archetypeName = "default"
	}

	archetype, ok := s.archetypes[archetypeName]
	if !ok {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("requested archetype does not exist"))
		return
	}

	e := archetype(s.c, r)

	// Override some properties according to query URL values.
	e.Title, _ = lo.Coalesce(query.Get("title"), e.Title)
	e.Description, _ = lo.Coalesce(query.Get("description"), e.Description)
	e.Reply, _ = lo.Coalesce(query.Get("reply"), e.Reply)
	e.Bookmark, _ = lo.Coalesce(query.Get("bookmark"), e.Bookmark)
	e.ID, _ = lo.Coalesce(query.Get("id"), e.ID)
	e.Content, _ = lo.Coalesce(query.Get("content"), e.Content)

	// Get stringified entry.
	str, err := e.String()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc, err := s.getTemplateDocument(r.URL.Path)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	archetypeNames := lo.Keys(s.archetypes)
	sort.Strings(archetypeNames)
	archetypesNode := doc.Find("eagle-archetype")
	for _, archetype := range archetypeNames {
		archetypesNode.Parent().AppendHtml(fmt.Sprintf(" <a href='?archetype=%s'>%s</a>", archetype, archetype))
	}
	archetypesNode.Remove()

	doc.Find(".eagle-editor input[name='id']").SetAttr("value", e.ID)
	doc.Find(".eagle-editor textarea[name='content']").SetText(str)

	s.serveDocument(w, r, doc, http.StatusOK)
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

	doc, err := s.getTemplateDocument(editPath)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc.Find("main input[name='id']").SetAttr("value", ee.ID)
	doc.Find("main input[name='rename']").SetAttr("value", ee.ID)
	doc.Find("main textarea[name='content']").SetText(str)

	if !ee.Date.IsZero() {
		doc.Find("main input[name='lastmod']").SetAttr("checked", "on")
	}

	s.serveDocument(w, r, doc, http.StatusOK)
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
