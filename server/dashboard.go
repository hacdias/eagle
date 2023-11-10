package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/hashicorp/go-multierror"
	"go.hacdias.com/indieauth"
)

const (
	dashboardPath = "/dashboard/"
)

type dashboardData struct {
	ActionSuccess bool
	Token         string
	MediaLocation string
}

func (s *Server) dashboardGet(w http.ResponseWriter, r *http.Request) {
	s.serveDashboard(w, r, &dashboardData{})
}

func (s *Server) dashboardPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	if r.Form.Get("action") != "" {
		s.dashboardPostAction(w, r)
		return
	} else if r.Form.Get("guestbook-action") != "" {
		s.dashboardPostGuestbook(w, r)
		return
	} else if r.Form.Get("token") == "true" {
		s.dashboardPostToken(w, r)
		return
	} else if err := r.ParseMultipartForm(20 << 20); err == nil {
		s.dashboardPostUpload(w, r)
		return
	}

	s.dashboardGet(w, r)
}

func (s *Server) dashboardPostAction(w http.ResponseWriter, r *http.Request) {
	actions := r.Form["action"]
	data := &dashboardData{}

	var errs *multierror.Error
	for _, actionName := range actions {
		if fn, ok := s.actions[actionName]; ok {
			errs = multierror.Append(errs, fn())
			data.ActionSuccess = true
		}
	}
	if err := errs.ErrorOrNil(); err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveDashboard(w, r, data)
}

func (s *Server) dashboardPostToken(w http.ResponseWriter, r *http.Request) {
	data := &dashboardData{}

	clientID := r.Form.Get("client_id")
	scope := r.Form.Get("scope")
	expiry, err := handleExpiry(r.Form.Get("expiry"))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("expiry param is invalid: %w", err))
	}

	if err := indieauth.IsValidClientIdentifier(clientID); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("invalid client_id: %w", err))
		return
	}

	signed, err := s.generateToken(clientID, scope, expiry)
	if err == nil {
		data.Token = signed
	} else {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveDashboard(w, r, data)
}

func (s *Server) dashboardPostUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		// NOTE: I'm not using http.DetectContentType because it depends
		// on OS specific mime type registries. Thus, it was being unreliable
		// on different OSes.
		contentType := header.Header.Get("Content-Type")
		mime := mimetype.Lookup(contentType)
		if mime.Is("application/octet-stream") {
			mime = mimetype.Detect(raw)
		}

		if mime == nil {
			s.serveErrorHTML(w, r, http.StatusBadRequest, err)
			return
		}

		ext = mime.Extension()
	}

	var location string

	if r.Form.Get("preserve-filename") == "on" {
		location, err = s.media.UploadMedia(strings.TrimSuffix(header.Filename, ext), ext, bytes.NewReader(raw))
	} else {
		location, err = s.media.UploadAnonymousMedia(ext, bytes.NewReader(raw))
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveDashboard(w, r, &dashboardData{
		MediaLocation: location,
	})
}

func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request, data *dashboardData) {
	doc, err := s.getTemplateDocument(r.URL.Path)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	actionTemplate := doc.Find("eagle-action").First().Children().First()
	doc.Find("eagle-actions").Empty()
	for _, action := range s.getActions() {
		node := actionTemplate.Clone()
		node.Find("input[name=action]").SetAttr("value", action)
		node.Find("action-name").ReplaceWithHtml(action)
		doc.Find("eagle-actions").AppendSelection(node)
	}

	if !data.ActionSuccess {
		doc.Find("eagle-action-success").Remove()
	}

	mediaNode := doc.Find("eagle-media-location")
	if data.MediaLocation != "" {
		mediaNode.Find("eagle-media-location-value").ReplaceWithHtml(data.MediaLocation)
		mediaNode.ReplaceWithSelection(mediaNode.Children())
	} else {
		mediaNode.Remove()
	}

	tokenNode := doc.Find("eagle-token")
	if data.Token != "" {
		tokenNode.Find("eagle-token-value").ReplaceWithHtml(data.Token)
		tokenNode.ReplaceWithSelection(tokenNode.Children())
	} else {
		tokenNode.Remove()
	}

	guestbookEntries, err := s.badger.GetGuestbookEntries(r.Context())
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, fmt.Errorf("error getting guestbook entries: %w", err))
		return
	}
	guestbookNode := doc.Find("eagle-guestbook-entries")
	if len(guestbookEntries) != 0 {
		guestbookEntryTemplate := doc.Find("eagle-guestbook-entry")
		guestbookNode.Empty()
		for _, e := range guestbookEntries {
			node := guestbookEntryTemplate.Clone()
			node.Find("eagle-guestbook-name").ReplaceWithHtml(e.Name)
			node.Find("eagle-guestbook-website").ReplaceWithHtml(e.Website)
			node.Find("eagle-guestbook-date").ReplaceWithHtml(e.Date.String())
			node.Find("eagle-guestbook-content").ReplaceWithHtml(e.Content)
			node.Find("input[name='guestbook-id']").SetAttr("value", e.ID)
			guestbookNode.AppendSelection(node.Children())
		}
		guestbookNode.ReplaceWithSelection(guestbookNode.Children())
	} else {
		guestbookNode.Remove()
	}

	s.serveDocument(w, r, doc, http.StatusOK)
}
