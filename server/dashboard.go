package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/hacdias/indieauth/v3"
	"github.com/hashicorp/go-multierror"
)

const (
	eaglePath = "/eagle"
)

type dashboardData struct {
	Actions       []string
	Success       bool
	Token         string
	MediaLocation string
}

func (s *Server) dashboardGet(w http.ResponseWriter, r *http.Request) {
	s.serveDashboard(w, r, &dashboardData{
		Actions: s.getActions(),
	})
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
	data := &dashboardData{
		Actions: s.getActions(),
	}

	var errs *multierror.Error
	for _, actionName := range actions {
		if fn, ok := s.actions[actionName]; ok {
			errs = multierror.Append(errs, fn())
			data.Success = true
		}
	}
	if err := errs.ErrorOrNil(); err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveDashboard(w, r, data)
}

func (s *Server) dashboardPostToken(w http.ResponseWriter, r *http.Request) {
	data := &dashboardData{
		Actions: s.getActions(),
	}

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

	fmt.Println(header.Filename)

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
		Actions:       s.getActions(),
		MediaLocation: location,
	})
}

func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request, data *dashboardData) {
	s.serveHTML(w, r, &renderData{
		Title: "Dashboard",
		Data:  data,
	}, templateDashboard, http.StatusOK)
}
