package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hacdias/eagle/eagle"
)

const dashboardPath = "/dashboard"

func (s *Server) redirectWithStatus(w http.ResponseWriter, status string) {
	s.renderDashboard(w, "status", &dashboardData{Content: status})
}

type rootPageData struct {
	Entries      []*eagle.SearchEntry
	Drafts       bool
	Query        string
	NextPage     string
	PreviousPage string
}

func (s *Server) dashboardGetHandler(w http.ResponseWriter, r *http.Request) {
	data := &dashboardData{}

	query, page, err := getSearchQuery(r)
	if err != nil {
		data.Content = err.Error()
		s.renderDashboard(w, "root", data)
		return
	}

	dd := &rootPageData{}

	if r.URL.Query().Get("drafts") == "on" {
		t := true
		query.Draft = &t
		dd.Drafts = true
	}

	query.ByDate = true
	entries, err := s.Search(query, page)
	if err != nil {
		data.Content = err.Error()
	}

	dd.Entries = entries
	dd.Query = query.Query

	if page > 0 {
		p := r.URL.Query()
		p.Set("p", strconv.Itoa(page-1))
		dd.PreviousPage = dashboardPath + "/?" + p.Encode()
	}

	n := r.URL.Query()
	n.Set("p", strconv.Itoa(page+1))
	dd.NextPage = dashboardPath + "/?" + n.Encode()

	data.Data = dd
	s.renderDashboard(w, "root", data)
}

func recentlyTemplate() (*eagle.Entry, string) {
	t := time.Now()
	month := t.Format("January")

	entry := &eagle.Entry{
		Content: "How was last month?",
		Metadata: eagle.Metadata{
			Draft: true,
			Title: fmt.Sprintf("Recently in %s '%s", month, t.Format("06")),
			Date:  t,
			Tags:  []string{"recently"},
		},
	}

	id := fmt.Sprintf("/articles/%s-%s/", strings.ToLower(month), t.Format("2006"))
	return entry, id
}

func defaultTemplate() (*eagle.Entry, string) {
	t := time.Now()

	entry := &eagle.Entry{
		Content: "Lorem ipsum...",
		Metadata: eagle.Metadata{
			Date: t,
			Tags: []string{"example"},
		},
	}

	id := fmt.Sprintf("micro/%s/SLUG", t.Format("2006/01"))
	return entry, id
}

func (s *Server) newGetHandler(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")

	var (
		entry *eagle.Entry
		id    string
	)

	switch template {
	case "recently":
		entry, id = recentlyTemplate()
	default:
		entry, id = defaultTemplate()
	}

	reply := sanitizeReplyURL(r.URL.Query().Get("reply"))
	if reply != "" {
		var err error
		entry.Metadata.ReplyTo, err = s.GetXRay(reply)
		if err != nil {
			s.dashboardError(w, r, err)
			return
		}
	}

	str, err := entry.String()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "new", &dashboardData{
		Content: str,
		ID:      id,
	})
}

func (s *Server) webmentionsGetHandler(w http.ResponseWriter, r *http.Request) {
	id, err := sanitizeID(chi.URLParam(r, "*"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.renderDashboard(w, "webmentions", &dashboardData{})
		return
	}

	entry, err := s.GetEntry(id)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	targets, _, _, err := s.GetWebmentionTargets(entry)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.renderDashboard(w, "webmentions", &dashboardData{Targets: targets, ID: id})
}

func (s *Server) editGetHandler(w http.ResponseWriter, r *http.Request) {
	id, err := sanitizeID(chi.URLParam(r, "*"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.renderDashboard(w, "edit", &dashboardData{})
		return
	}

	entry, err := s.GetEntry(id)
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
		Data: map[string]interface{}{
			"Entry":   entry,
			"Content": str,
		},
	})
}

func (s *Server) replyGetHandler(w http.ResponseWriter, r *http.Request) {
	s.renderDashboard(w, "reply", &dashboardData{})
}

func (s *Server) blogrollGetHandler(w http.ResponseWriter, r *http.Request) {
	if s.Miniflux == nil {
		s.dashboardError(w, r, errors.New("miniflux integration is disabled"))
		return
	}

	feeds, err := s.Miniflux.Fetch()
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

	data, err := s.ReadFile(path)
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

	err = s.Persist(path, []byte(content), "edit: update "+path)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	err = s.Build(true)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	s.redirectWithStatus(w, path+" updated! 🗄")
}

func (s *Server) syncGetHandler(w http.ResponseWriter, r *http.Request) {
	_, err := s.Sync()
	if err != nil {
		s.dashboardError(w, r, err)
	} else {
		s.redirectWithStatus(w, "Sync was successfull! ⚡️")
	}
}

func (s *Server) buildGetHandler(w http.ResponseWriter, r *http.Request) {
	clean := r.URL.Query().Get("mode") == "clean"
	err := s.Build(clean)
	if err != nil {
		s.dashboardError(w, r, err)
	} else {
		s.redirectWithStatus(w, "Build was successfull! 💪")
	}
}

func (s *Server) rebuildIndexGetHandler(w http.ResponseWriter, r *http.Request) {
	err := s.RebuildIndex()
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

	entry, err := s.GetEntry(id)
	if err != nil {
		s.NotifyError(err)
		return
	}

	s.goWebmentions(entry)
	s.redirectWithStatus(w, "Webmentions scheduled! 💭")
}

func (s *Server) newPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	content := r.FormValue("content")
	twitter := r.FormValue("twitter") == "on"

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

	entry, err := s.ParseEntry(id, content)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	err = s.newEditPostSaver(entry, false, twitter)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if entry.Metadata.Draft {
		s.redirectWithStatus(w, entry.ID+" updated successfullyl! ⚡️")
		return
	}

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
	action := r.FormValue("action")
	twitter := r.FormValue("twitter") == "on"

	id, err := sanitizeID(chi.URLParam(r, "*"))
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if id == "" {
		s.dashboardError(w, r, errors.New("no ID provided"))
		return
	}

	_, err = s.GetEntry(id)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	entry, err := s.ParseEntry(id, content)
	if err != nil {
		s.dashboardError(w, r, err)
		return
	}

	if lastmod == "on" {
		entry.Metadata.Lastmod = time.Now()
	}

	switch action {
	case "delete":
		entry.Metadata.ExpiryDate = time.Now()
	case "undelete":
		entry.Metadata.ExpiryDate = time.Time{}
	case "publish":
		entry.Metadata.Draft = false
	case "update":
		// Nothing else.
	}

	err = s.newEditPostSaver(entry, action == "delete", twitter)
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

func (s *Server) newEditPostSaver(entry *eagle.Entry, clean, twitter bool) error {
	err := s.SaveEntry(entry)
	if err != nil {
		return err
	}

	err = s.Build(clean)
	if err != nil {
		return err
	}

	if entry.Metadata.Draft {
		return nil
	}

	go func() {
		s.goWebmentions(entry)

		if twitter {
			s.goSyndicate(entry)
		}
	}()

	return nil
}

func (s *Server) dashboardError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	s.renderDashboard(w, "error", &dashboardData{
		Content: err.Error(),
	})
}
