package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hacdias/eagle/eagle"
)

func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	s.renderDashboard(w, "dashboard", &dashboardData{
		Reply:  r.URL.Query().Get("reply"),
		Edit:   r.URL.Query().Get("edit"),
		Delete: r.URL.Query().Get("delete"),
	})
}

func (s *Server) editorGetHandler(w http.ResponseWriter, r *http.Request) {
	id, err := sanitizeID(r.URL.Query().Get("id"))
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	reply := cleanReplyURL(r.URL.Query().Get("reply"))

	if (reply != "") && id != "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Cannot set id and reply at the same time.\n"))
		return
	}

	var entry *eagle.Entry

	if id != "" {
		entry, err = s.e.GetEntry(id)
	} else {
		entry = &eagle.Entry{
			Content: "Lorem ipsum...",
			Metadata: eagle.EntryMetadata{
				Date: time.Now(),
				Tags: []string{"example"},
			},
		}
	}

	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	if reply != "" {
		entry.Metadata.Title = ""
		entry.Metadata.ReplyTo, err = s.e.Crawl(reply)
		if err != nil {
			s.serveError(w, http.StatusInternalServerError, err)
			return
		}
	}

	str, err := s.e.EntryToString(entry)
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	s.renderDashboard(w, "editor", &dashboardData{
		Content:   str,
		ID:        id,
		DefaultID: fmt.Sprintf("micro/%s/SLUG", time.Now().Format("2006/01")),
	})
}

func (s *Server) editorPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
	}

	if r.FormValue("id") == "" {
		s.serveError(w, http.StatusBadRequest, errors.New("id is missing"))
		return
	}

	var code int
	switch r.FormValue("action") {
	case "create", "update":
		code, err = s.editorUpdate(w, r)
	case "delete":
		code, err = s.editorDelete(w, r)
	default:
		s.serveError(w, http.StatusBadRequest, errors.New("invalid action type"))
		return
	}

	if code >= 200 && code < 400 {
		w.WriteHeader(code)
	} else if code >= 400 {
		s.Errorf("editor: post handler: %s", err)
		s.serveError(w, code, err)
	}
}

func (s *Server) editorUpdate(w http.ResponseWriter, r *http.Request) (int, error) {
	action := r.FormValue("action")
	content := r.FormValue("content")
	twitter := r.FormValue("twitter")
	id, err := sanitizeID(r.FormValue("id"))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if id == r.FormValue("defaultid") {
		return http.StatusBadRequest, errors.New("id must be updated")
	}

	entry, err := s.e.ParseEntry(id, content)
	if err != nil {
		return http.StatusBadRequest, err
	}

	s.e.PopulateMentions(entry)

	err = s.e.SaveEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.e.Build(false)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		s.sendWebmentions(entry)
		s.activity(entry)

		if action == "create" && twitter == "on" && s.e.Twitter != nil {
			s.syndicate(entry)
		}
	}()

	http.Redirect(w, r, entry.ID, http.StatusTemporaryRedirect)
	return 0, nil
}

func (s *Server) editorDelete(w http.ResponseWriter, r *http.Request) (int, error) {
	id := r.FormValue("id")

	entry, err := s.e.GetEntry(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.e.DeleteEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.e.Build(true)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	http.Redirect(w, r, entry.ID, http.StatusTemporaryRedirect)
	return 0, nil
}

func (s *Server) syndicate(entry *eagle.Entry) {
	if s.e.Twitter == nil {
		return
	}

	url, err := s.e.Twitter.Syndicate(entry)
	if err != nil {
		s.Errorf("failed to syndicate: %s", err)
		s.e.NotifyError(err)
		return
	}

	entry.Metadata.Syndication = append(entry.Metadata.Syndication, url)
	err = s.e.SaveEntry(entry)
	if err != nil {
		s.Errorf("failed to save entry: %s", err)
		s.e.NotifyError(err)
		return
	}

	err = s.e.Build(false)
	if err != nil {
		s.Errorf("failed to build: %s", err)
		s.e.NotifyError(err)
	}
}

func (s *Server) activity(entry *eagle.Entry) {
	s.staticFsLock.RLock()
	activity, err := s.staticFs.readAS2(entry.ID)
	s.staticFsLock.RUnlock()
	if err != nil {
		s.Errorf("coult not fetch activity for %s: %s", entry.ID, err)
		return
	}

	err = s.e.ActivityPub.PostFollowers(activity)
	if err != nil {
		s.Errorf("could not queue activity posting for %s: %s", entry.ID, err)
		return
	}

	s.Infof("activity posting for %s scheduled", entry.ID)
}

func (s *Server) sendWebmentions(entry *eagle.Entry) {
	var err error
	defer func() {
		if err != nil {
			s.e.NotifyError(err)
			s.Warnf("webmentions: %s", err)
		}
	}()

	s.staticFsLock.RLock()
	html, err := s.staticFs.readHTML(entry.ID)
	s.staticFsLock.RUnlock()
	if err != nil {
		s.Errorf("could not fetch HTML for %s: %v", entry.ID, err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return
	}

	targets := []string{}

	if entry.Metadata.ReplyTo != nil && entry.Metadata.ReplyTo.URL != "" {
		targets = append(targets, entry.Metadata.ReplyTo.URL)
	}

	doc.Find(".h-entry .e-content a").Each(func(i int, q *goquery.Selection) {
		val, ok := q.Attr("href")
		if !ok {
			return
		}

		u, err := url.Parse(val)
		if err != nil {
			targets = append(targets, val)
			return
		}

		base, err := url.Parse(entry.Permalink)
		if err != nil {
			targets = append(targets, val)
			return
		}

		targets = append(targets, base.ResolveReference(u).String())
	})

	s.Infow("webmentions: found targets", "entry", entry.ID, "permalink", entry.Permalink, "targets", targets)
	err = s.e.SendWebmention(entry.Permalink, targets...)
}

func cleanReplyURL(iu string) string {
	if strings.HasPrefix(iu, "https://twitter.com") && strings.Contains(iu, "/status/") {
		u, err := url.Parse(iu)
		if err != nil {
			return iu
		}

		for k := range u.Query() {
			u.Query().Del(k)
		}

		return u.String()
	}

	return iu
}

func sanitizeID(id string) (string, error) {
	if id != "" {
		u, err := url.Parse(id)
		if err != nil {
			return "", err
		}
		id = u.Path
	}
	return path.Clean(id), nil
}
