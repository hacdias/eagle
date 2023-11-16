package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/indielib/indieauth"
)

const (
	panelPath          = "/panel/"
	panelGuestbookPath = panelPath + "guestbook/"
	panelTokensPath    = panelPath + "tokens/"
)

type panelPage struct {
	Actions       []string
	ActionSuccess bool
	Token         string
	MediaLocation string
}

func (s *Server) panelGet(w http.ResponseWriter, r *http.Request) {
	s.servePanel(w, r, &panelPage{})
}

func (s *Server) panelPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	if r.Form.Get("action") != "" {
		s.panelPostAction(w, r)
		return
	} else if err := r.ParseMultipartForm(20 << 20); err == nil {
		s.panelPostUpload(w, r)
		return
	}

	s.panelGet(w, r)
}

func (s *Server) panelPostAction(w http.ResponseWriter, r *http.Request) {
	actions := r.Form["action"]
	data := &panelPage{}

	var err error
	for _, actionName := range actions {
		if fn, ok := s.actions[actionName]; ok {
			err = errors.Join(err, fn())
			data.ActionSuccess = true
		}
	}
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.buildNotify(false)
	s.servePanel(w, r, data)
}

func (s *Server) panelPostUpload(w http.ResponseWriter, r *http.Request) {
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

	s.servePanel(w, r, &panelPage{
		MediaLocation: location,
	})
}

func (s *Server) servePanel(w http.ResponseWriter, r *http.Request, data *panelPage) {
	data.Actions = s.getActions()
	s.renderTemplateWithContent(w, r, "panel.html", &pageData{
		Title: "Panel",
		Data:  data,
	})
}

func (s *Server) panelGuestbookGet(w http.ResponseWriter, r *http.Request) {
	guestbookEntries, err := s.badger.GetGuestbookEntries(r.Context())
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, fmt.Errorf("error getting guestbook entries: %w", err))
		return
	}

	s.renderTemplateWithContent(w, r, "panel-guestbook.html", &pageData{
		Title: "Panel Guestbook",
		Data:  guestbookEntries,
	})
}

func (s *Server) panelGuestbookPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	action := r.Form.Get("action")
	id := r.Form.Get("id")

	switch action {
	case "approve":
		e, err := s.badger.GetGuestbookEntry(r.Context(), id)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}

		entries := core.GuestbookEntries{}
		if err := s.fs.ReadJSON(guestbookFilename, &entries); err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, fmt.Errorf("error reading guestbook file: %w", err))
			return
		}
		entries = append(entries, e)
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Date.After(entries[j].Date)
		})
		message := "guestbook: new entry"
		if e.Name != "" {
			message += " from " + e.Name
		}
		if err := s.fs.WriteJSON(guestbookFilename, entries, message); err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, fmt.Errorf("error writing guestbook file: %w", err))
			return
		}

		go func() {
			_ = s.hugo.Build(false)
		}()

		fallthrough
	case "delete":
		err := s.badger.DeleteGuestbookEntry(r.Context(), id)
		if err != nil {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}
	default:
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("invalid action: %s", action))
		return
	}

	http.Redirect(w, r, r.URL.Path, http.StatusFound)
}

type tokenPage struct {
	Token string
}

func (s *Server) panelTokensGet(w http.ResponseWriter, r *http.Request) {
	s.renderTemplateWithContent(w, r, "panel-tokens.html", &pageData{
		Title: "Panel Tokens",
		Data:  &tokenPage{},
	})
}

func (s *Server) panelTokensPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	data := &tokenPage{}

	clientID := r.Form.Get("client_id")
	scope := r.Form.Get("scope")
	expiry, err := handleExpiry(r.Form.Get("expiry"))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("expiry param is invalid: %w", err))
		return
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

	s.renderTemplateWithContent(w, r, "panel-tokens.html", &pageData{
		Title: "Panel Tokens",
		Data:  data,
	})
}
