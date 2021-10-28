package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hacdias/eagle/eagle"
)

const dashboardPath = "/dashboard"

func (s *Server) redirectWithStatus(w http.ResponseWriter, status string) {
	s.renderDashboard(w, "status", &dashboardData{Content: status})
}

func (s *Server) dashboardGetHandler(w http.ResponseWriter, r *http.Request) {
	data := &dashboardData{}

	query, page, err := getSearchQuery(r)
	if err != nil {
		data.Content = err.Error()
		s.renderDashboard(w, "root", data)
		return
	}

	if r.URL.Query().Get("drafts") == "on" {
		t := true
		query.Draft = &t
		data.Drafts = true
	}

	query.ByDate = true
	entries, err := s.e.Search(query, page)
	if err != nil {
		data.Content = err.Error()
	}

	data.Entries = entries
	data.Query = query.Query

	if page > 0 {
		p := r.URL.Query()
		p.Set("p", strconv.Itoa(page-1))
		data.PreviousPage = dashboardPath + "/?" + p.Encode()
	}

	n := r.URL.Query()
	n.Set("p", strconv.Itoa(page+1))
	data.NextPage = dashboardPath + "/?" + n.Encode()

	s.renderDashboard(w, "root", data)
}

func (s *Server) newGetHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: add option for different types? Archetypes?
	entry := &eagle.Entry{
		Content: "Lorem ipsum...",
		Metadata: eagle.EntryMetadata{
			Date: time.Now(),
			Tags: []string{"example"},
		},
	}

	str, err := entry.String()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "new", &dashboardData{
		Content: str,
		ID:      fmt.Sprintf("micro/%s/SLUG", time.Now().Format("2006/01")),
	})
}

func (s *Server) webmentionsGetHandler(w http.ResponseWriter, r *http.Request) {
	id, err := sanitizeID(r.URL.Query().Get("url"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.renderDashboard(w, "webmentions", &dashboardData{})
		return
	}

	entry, err := s.e.GetEntry(id)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	targets, err := s.getWebmentionTargets(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "webmentions", &dashboardData{Targets: targets, ID: id})
}

func (s *Server) editGetHandler(w http.ResponseWriter, r *http.Request) {
	id, err := sanitizeID(r.URL.Query().Get("url"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.renderDashboard(w, "edit", &dashboardData{})
		return
	}

	entry, err := s.e.GetEntry(id)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	str, err := entry.String()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "edit", &dashboardData{
		ID:      entry.ID,
		Content: str,
	})
}

func (s *Server) replyGetHandler(w http.ResponseWriter, r *http.Request) {
	reply := sanitizeReplyURL(r.URL.Query().Get("url"))
	if reply == "" {
		s.renderDashboard(w, "reply", &dashboardData{})
		return
	}

	entry := &eagle.Entry{
		Content: "Your reply here...",
		Metadata: eagle.EntryMetadata{
			Date: time.Now(),
			Tags: []string{"example"},
		},
	}

	var err error
	entry.Metadata.ReplyTo, err = s.e.Crawl(reply)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	str, err := entry.String()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "reply", &dashboardData{
		Content: str,
		ID:      fmt.Sprintf("micro/%s/SLUG", time.Now().Format("2006/01")),
	})
}

func (s *Server) deleteGetHandler(w http.ResponseWriter, r *http.Request) {
	id, err := sanitizeID(r.URL.Query().Get("url"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.renderDashboard(w, "delete", &dashboardData{})
		return
	}

	entry, err := s.e.GetEntry(id)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	str, err := entry.String()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "delete", &dashboardData{
		ID:      entry.ID,
		Content: str,
	})
}

func (s *Server) blogrollGetHandler(w http.ResponseWriter, r *http.Request) {
	if s.e.Miniflux == nil {
		s.dashboardError(w, r, errors.New("miniflux integration is disabled"))
		return
	}

	feeds, err := s.e.Miniflux.Fetch()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	data, err := json.MarshalIndent(feeds, "", "  ")
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "gedit", &dashboardData{
		ID:      "data/blogroll.json",
		Content: string(data),
	})
}

func (s *Server) geditGetHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	path := r.FormValue("path")
	if path == "" {
		s.renderDashboard(w, "gedit", &dashboardData{})
		return
	}

	data, err := s.e.ReadFile(path)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "gedit", &dashboardData{
		ID:      path,
		Content: string(data),
	})
}

func (s *Server) geditPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	path := r.FormValue("path")
	if path == "" {
		s.dashboardError(w, r, errors.New("no path provided"))
		return
	}

	content := r.FormValue("content")
	if content == "" {
		s.dashboardError(w, r, errors.New("no content provided"))
		return
	}

	err = s.e.Persist(path, []byte(content), "edit: update "+path)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	err = s.e.Build(true)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.redirectWithStatus(w, path+" updated! 🗄")
}

func (s *Server) syncGetHandler(w http.ResponseWriter, r *http.Request) {
	_, err := s.e.Sync()
	if err != nil {
		s.dashboardError(w, r, err)
	} else {
		s.redirectWithStatus(w, "Sync was successfull! ⚡️")
	}
}

func (s *Server) buildGetHandler(w http.ResponseWriter, r *http.Request) {
	clean := r.URL.Query().Get("mode") == "clean"
	err := s.e.Build(clean)
	if err != nil {
		s.dashboardError(w, r, err)
	} else {
		s.redirectWithStatus(w, "Build was successfull! 💪")
	}
}

func (s *Server) rebuildIndexGetHandler(w http.ResponseWriter, r *http.Request) {
	err := s.e.RebuildIndex()
	if err != nil {
		s.dashboardError(w, r, err)
	} else {
		s.redirectWithStatus(w, "Search index rebuilt! 🔎")
	}
}

func (s *Server) webmentionsPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	id, err := sanitizeID(r.FormValue("url"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	entry, err := s.e.GetEntry(id)
	if err != nil {
		s.e.NotifyError(err)
		return
	}

	s.goWebmentions(entry)
	s.redirectWithStatus(w, "Webmentions scheduled! 💭")
}

func (s *Server) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	id, err := sanitizeID(r.FormValue("url"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.dashboardError(w, r, errors.New("no ID provided"))
		return
	}

	entry, err := s.e.GetEntry(id)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	err = s.e.DeleteEntry(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	err = s.e.Build(true)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	http.Redirect(w, r, entry.Permalink, http.StatusTemporaryRedirect)
}

func (s *Server) newPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	content := r.FormValue("content")
	twitter := r.FormValue("twitter")

	id, err := sanitizeID(r.FormValue("id"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.dashboardError(w, r, errors.New("no ID provided"))
		return
	}

	if id == r.FormValue("defaultid") {
		s.dashboardError(w, r, errors.New("cannot use default ID"))
		return
	}

	entry, err := s.e.ParseEntry(id, content)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	err = s.newEditPostSaver(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if entry.Metadata.Draft {
		s.redirectWithStatus(w, entry.ID+" updated successfullyl! ⚡️")
		return
	}

	go func() {
		if twitter == "on" {
			s.goSyndicate(entry)
		}
	}()

	http.Redirect(w, r, entry.Permalink, http.StatusTemporaryRedirect)
}

func (s *Server) editPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	content := r.FormValue("content")
	lastmod := r.FormValue("lastmod")

	id, err := sanitizeID(r.FormValue("id"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.dashboardError(w, r, errors.New("no ID provided"))
		return
	}

	_, err = s.e.GetEntry(id)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	entry, err := s.e.ParseEntry(id, content)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if lastmod == "on" {
		entry.Metadata.Lastmod = time.Now()
	}

	err = s.newEditPostSaver(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if entry.Metadata.Draft {
		s.redirectWithStatus(w, entry.ID+" updated successfullyl! ⚡️")
	} else {
		http.Redirect(w, r, entry.Permalink, http.StatusTemporaryRedirect)
	}
}

func (s *Server) dashboardError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	s.renderDashboard(w, "error", &dashboardData{
		Content: err.Error(),
	})
}

func (s *Server) newEditPostSaver(entry *eagle.Entry) error {
	err := s.e.SaveEntry(entry)
	if err != nil {
		return err
	}

	err = s.e.Build(false)
	if err != nil {
		return err
	}

	if entry.Metadata.Draft {
		return nil
	}

	go func() {
		s.goWebmentions(entry)
	}()

	return nil
}
