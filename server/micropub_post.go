package server

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hacdias/eagle/middleware/micropub"
	"github.com/hacdias/eagle/services"
)

func (s *Server) postMicropubHandler(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()

	mr, err := micropub.ParseRequest(r)
	if err != nil {
		s.serveError(w, http.StatusBadRequest, err)
		return
	}

	var code int

	switch mr.Action {
	case micropub.ActionCreate:
		code, err = s.micropubCreate(w, r, mr)
	case micropub.ActionUpdate:
		code, err = s.micropubUpdate(w, r, mr)
	case micropub.ActionDelete:
		code, err = s.micropubRemove(w, r, mr)
	case micropub.ActionUndelete:
		code, err = s.micropubUnremove(w, r, mr)
	default:
		code, err = http.StatusBadRequest, errors.New("invalid action")
	}

	if code >= 200 && code < 400 {
		w.WriteHeader(code)
	} else if code >= 400 {
		s.Errorf("micropub: error on post: %s", err)
		s.serveError(w, code, err)
	}

	if mr.Action == micropub.ActionCreate {
		return
	}

	err = s.Hugo.Build(mr.Action == micropub.ActionDelete)
	if err != nil {
		s.Errorf("micropub: error hugo build: %s", err)
		s.Notify.Error(err)
	}
}

func (s *Server) micropubCreate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	s.Debug("micropub: create request received")
	entry, synd, err := s.Hugo.FromMicropub(mr)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.Hugo.SaveEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	for _, rel := range synd.Related {
		err = s.XRay.RequestAndSave(rel)
		if err != nil {
			s.Warnf("could not xray %s: %s", rel, err)
			s.Notify.Error(err)
		}
	}

	err = s.Store.Persist("add " + entry.ID)
	if err != nil {
		s.Errorf("micropub: error git commit: %s", err)
		s.Notify.Error(err)
	}

	err = s.Hugo.Build(false)
	if err != nil {
		s.Errorf("micropub: error hugo build: %s", err)
		s.Notify.Error(err)
	}

	url := s.c.Domain + entry.ID
	http.Redirect(w, r, url, http.StatusAccepted)

	go func() {
		s.sendWebmentions(entry)
		s.syndicate(entry, synd)
	}()

	s.Debug("micropub: create request ok")
	return 0, nil
}

func (s *Server) micropubUpdate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	s.Debug("micropub: update request received")
	id := strings.Replace(mr.URL, s.c.Domain, "", 1)
	entry, err := s.Hugo.GetEntry(id)
	if err != nil {
		s.Errorf("micropub: cannot get entry: %s", err)
		return http.StatusBadRequest, err
	}

	err = entry.Update(mr)
	if err != nil {
		s.Errorf("micropub: cannot update entry: %s", err)
		return http.StatusBadRequest, err
	}

	err = s.Hugo.SaveEntry(entry)
	if err != nil {
		s.Errorf("micropub: cannot save entry: %s", err)
		return http.StatusInternalServerError, err
	}

	err = s.Store.Persist("update " + entry.ID)
	if err != nil {
		s.Errorf("micropub: cannot git commit: %s", err)
		return http.StatusInternalServerError, err
	}

	http.Redirect(w, r, mr.URL, http.StatusOK)
	s.Debug("micropub: update request ok")
	go s.sendWebmentions(entry)
	return 0, nil
}

func (s *Server) micropubUnremove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	s.Debug("micropub: unremove request received")
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	entry, err := s.Hugo.GetEntry(id)
	if err != nil {
		return http.StatusBadRequest, err
	}

	delete(entry.Metadata, "expiryDate")

	err = s.Hugo.SaveEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.Store.Persist("undelete " + id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	s.Debug("micropub: unremove request ok")
	return http.StatusOK, nil
}

func (s *Server) micropubRemove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	s.Debug("micropub: remove request received")
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	entry, err := s.Hugo.GetEntry(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	entry.Metadata["expiryDate"] = time.Now().Format(time.RFC3339)

	err = s.Hugo.SaveEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.Store.Persist("delete " + id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	s.Debug("micropub: remove request ok")
	return http.StatusOK, nil
}

func (s *Server) syndicate(entry *services.HugoEntry, synd *services.Syndication) {
	syndication, err := s.Syndicator.Syndicate(entry, synd)
	if err != nil {
		s.Errorf("gossip: failed to syndicate: %s", err)
		s.Notify.Error(err)
		return
	}

	s.Lock()

	defer func() {
		s.Unlock()
		if err != nil {
			s.Errorf("gossip: %s", err)
			s.Notify.Error(err)
		}
	}()

	entry, err = s.Hugo.GetEntry(entry.ID)
	if err != nil {
		s.Errorf("gossip: failed to get entry: %s", err)
		return
	}

	props := entry.Metadata["properties"].(map[string][]interface{})
	props["syndication"] = []interface{}{}

	for _, s := range syndication {
		props["syndication"] = append(props["syndication"], s)
	}

	entry.Metadata["properties"] = props

	err = s.Hugo.SaveEntry(entry)
	if err != nil {
		return
	}

	err = s.Store.Persist("syndication on " + entry.ID)
	if err != nil {
		return
	}

	err = s.Hugo.Build(false)
}

func (s *Server) sendWebmentions(entry *services.HugoEntry) {
	var err error
	defer func() {
		if err != nil {
			s.Notify.Error(err)
			s.Warnf("webmentions: %s", err)
		}
	}()

	s.Debug("webmentions: entered")

	s.Lock()
	html, err := s.Hugo.GetEntryHTML(entry.ID)
	if err != nil {
		s.Unlock()
		return
	}
	s.Unlock()

	s.Debugw("webmentions: got HTML", "entry", entry.ID)

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return
	}

	targets := []string{}

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

	s.Debugw("webmentions: found targets %s", "entry", entry.ID, "targets", targets)
	err = s.Webmentions.Send(entry.Permalink, targets...)
}
