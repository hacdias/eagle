package server

import (
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
		// s.sendWebmentions(entry)
		// s.activity(entry)

		if action == "create" {
			// TODO .syndicate(entry, synd)
		}

		/*
			err := s.MeiliSearch.Add(entry)
			if err != nil {
				s.Warnf("could not add to meilisearch: %s", err)
				s.NotifyError(err)
			}
		*/

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

	go func() {
		// TODO err := s.MeiliSearch.Delete(entry)
	}()

	http.Redirect(w, r, entry.ID, http.StatusTemporaryRedirect)
	return 0, nil
}

/*

func (s *Server) syndicate(entry *services.HugoEntry, synd *services.Syndication) {
	s.Debug("syndicate: started")
	syndication, err := s.Syndicator.Syndicate(entry, synd)
	if err != nil {
		s.Errorf("syndicate: failed to syndicate: %s", err)
		s.NotifyError(err)
		return
	}

	s.Debugw("syndicate: got syndication results", "syndication", syndication)
	s.Lock()
	defer func() {
		s.Unlock()
		if err != nil {
			s.Errorf("syndicate: %s", err)
			s.NotifyError(err)
		}
	}()

	s.Debug("syndicate: fetch hugo entry")
	entry, err = s.Hugo.GetEntry(entry.ID)
	if err != nil {
		return
	}
	s.Debug("syndicate: got hugo entry")

	props := entry.Metadata["properties"].(map[string][]interface{})
	props["syndication"] = []interface{}{}

	for _, s := range syndication {
		props["syndication"] = append(props["syndication"], s)
	}

	entry.Metadata["properties"] = props

	s.Debug("syndicate: saving hugo entry")
	err = s.Hugo.SaveEntry(entry)
	if err != nil {
		return
	}
	s.Debug("syndicate: hugo entry saved")

	err = s.Store.Persist("syndication on " + entry.ID)
	if err != nil {
		return
	}

	err = s.Hugo.Build(false)
}

*/

func (s *Server) activity(entry *eagle.Entry) {
	activity, err := s.getAS2(entry.ID)
	if err != nil {
		s.Errorf("coult not fetch activity for %s: %s", entry.ID, err)
		return
	}

	err = s.ActivityPub.PostFollowers(activity)
	if err != nil {
		s.Errorf("coult not post activity %s: %s", entry.ID, err)
		return
	}

	s.Infof("activity %s scheduled for sending", entry.ID)
}

func (s *Server) sendWebmentions(entry *eagle.Entry) {
	var err error
	defer func() {
		if err != nil {
			s.NotifyError(err)
			s.Warnf("webmentions: %s", err)
		}
	}()

	s.Debug("webmentions: entered")

	s.Lock()
	reader := s.getHTML(entry.ID)
	if reader == nil {
		err = fmt.Errorf("could not get reader for %s", entry.ID)
		s.Unlock()
		return
	}

	s.Debugw("webmentions: got HTML", "entry", entry.ID)

	doc, err := goquery.NewDocumentFromReader(reader)
	s.Unlock()
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

	s.Debugw("webmentions: found targets", "entry", entry.ID, "permalink", entry.Permalink, "targets", targets)
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
