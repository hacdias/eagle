package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hacdias/eagle/eagle"
)

func (s *Server) dashboardGetHandler(w http.ResponseWriter, r *http.Request) {
	s.renderDashboard(w, "dashboard", &dashboardData{})
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

	str, err := s.e.EntryToString(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "new", &dashboardData{
		Content: str,
		ID:      fmt.Sprintf("micro/%s/SLUG", time.Now().Format("2006/01")),
	})
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

	str, err := s.e.EntryToString(entry)
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

	str, err := s.e.EntryToString(entry)
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

	str, err := s.e.EntryToString(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "delete", &dashboardData{
		ID:      entry.ID,
		Content: str,
	})
}

func (s *Server) dashboardPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if r.FormValue("sync") == "true" {
		err := s.e.Sync()
		if err != nil {
			s.dashboardError(w, r, err)
		} else {
			s.renderDashboard(w, "dashboard", &dashboardData{Content: "Sync was successfull! ⚡️"})
		}
		return
	}

	if r.FormValue("build") == "true" {
		clean := r.FormValue("mode") == "clean"
		err := s.e.Build(clean)
		if err != nil {
			s.dashboardError(w, r, err)
		} else {
			s.renderDashboard(w, "dashboard", &dashboardData{Content: "Build was successfull! 💪"})
		}
		return
	}

	if r.FormValue("rebuild-index") == "true" {
		err = s.e.RebuildIndex()
		if err != nil {
			s.dashboardError(w, r, err)
		} else {
			s.renderDashboard(w, "dashboard", &dashboardData{Content: "Search index rebuilt! 🔎"})
		}
		return
	}

	reshare := r.FormValue("reshare")
	if reshare != "" {
		id, err := sanitizeID(reshare)
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
		s.goActivity(entry)
		s.renderDashboard(w, "dashboard", &dashboardData{Content: "Webmentions and Activity scheduled! 💭"})
		return
	}

	s.renderDashboard(w, "dashboard", &dashboardData{})
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

	http.Redirect(w, r, entry.ID, http.StatusTemporaryRedirect)
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

	err = s.e.Build(false)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	err = s.newEditPostSaver(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	go func() {
		if twitter == "on" {
			s.goSyndicate(entry)
		}
	}()

	http.Redirect(w, r, entry.ID, http.StatusTemporaryRedirect)
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

	err = s.e.Build(false)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	err = s.newEditPostSaver(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	http.Redirect(w, r, entry.ID, http.StatusTemporaryRedirect)
}

func (s *Server) dashboardError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	s.renderDashboard(w, "error", &dashboardData{
		Content: err.Error(),
	})
}

func (s *Server) newEditPostSaver(entry *eagle.Entry) error {
	s.e.PopulateMentions(entry)

	err := s.e.SaveEntry(entry)
	if err != nil {
		return err
	}

	err = s.e.Build(false)
	if err != nil {
		return err
	}

	go func() {
		s.goWebmentions(entry)
		s.goActivity(entry)
	}()

	return nil
}
