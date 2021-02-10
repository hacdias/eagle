package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	id := r.URL.Query().Get("id")
	if id != "" {
		u, err := url.Parse(id)
		if err != nil {
			s.serveError(w, http.StatusInternalServerError, err)
			return
		}
		id = u.Path
	}

	reply := cleanReplyURL(r.URL.Query().Get("reply"))

	if (reply != "") && id != "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Cannot set id and reply at the same time.\n"))
		return
	}

	var (
		entry *eagle.Entry
		err   error
	)

	if id != "" {
		entry, err = s.GetEntry(id)
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
		entry.Metadata.ReplyTo, err = s.Crawl(reply)
		if err != nil {
			s.serveError(w, http.StatusInternalServerError, err)
			return
		}
	}

	str, err := s.EntryToString(entry)
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
	r.ParseForm()

	if r.FormValue("id") == "" {
		s.serveError(w, http.StatusBadRequest, errors.New("id is missing"))
		return
	}

	var code int
	var err error

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
	id := r.FormValue("id")
	content := r.FormValue("content")
	twitter := r.FormValue("twitter")

	if id == r.FormValue("defaultid") {
		return http.StatusBadRequest, errors.New("id must be updated")
	}

	entry, err := s.ParseEntry(id, content)
	if err != nil {
		return http.StatusBadRequest, err
	}

	s.PopulateMentions(entry)

	err = s.SaveEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.Build(false)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go func() {
		// TODO s.sendWebmentions(entry)
		// s.activity(entry)

		if action == "create" && twitter == "on" && s.Twitter != nil {
			s.syndicate(entry)
		}
	}()

	http.Redirect(w, r, entry.ID, http.StatusTemporaryRedirect)
	return 0, nil
}

func (s *Server) editorDelete(w http.ResponseWriter, r *http.Request) (int, error) {
	id := r.FormValue("id")

	entry, err := s.GetEntry(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.DeleteEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.Build(true)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	http.Redirect(w, r, entry.ID, http.StatusTemporaryRedirect)
	return 0, nil
}

func (s *Server) syndicate(entry *eagle.Entry) {
	if s.Twitter == nil {
		return
	}

	url, err := s.Twitter.Syndicate(entry)
	if err != nil {
		s.Errorf("failed to syndicate: %s", err)
		s.NotifyError(err)
		return
	}

	entry.Metadata.Syndication = append(entry.Metadata.Syndication, url)
	err = s.SaveEntry(entry)
	if err != nil {
		s.Errorf("failed to save entry: %s", err)
		s.NotifyError(err)
		return
	}

	err = s.Build(false)
	if err != nil {
		s.Errorf("failed to build: %s", err)
		s.NotifyError(err)
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

	err = s.ActivityPub.PostFollowers(activity)
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
			s.NotifyError(err)
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
	err = s.SendWebmention(entry.Permalink, targets...)
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
