package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/pkg/micropub"
	"github.com/samber/lo"
)

func (s *Server) micropubGet(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("q") {
	case "source":
		s.micropubSource(w, r)
	case "config", "syndicate-to":
		s.micropubConfig(w, r)
	default:
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	}
}

func (s *Server) micropubSource(w http.ResponseWriter, r *http.Request) {
	id, err := s.micropubParseURL(r.URL.Query().Get("url"))
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "The request is missing the URL.")
		return
	}

	entry, err := s.fs.GetEntry(id)
	if err != nil {
		if os.IsNotExist(err) {
			s.serveErrorJSON(w, http.StatusNotFound, "invalid_request", fmt.Sprintf("Post cannot be found: %s.", id))
		} else {
			s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
		}
		return
	}

	s.serveJSON(w, http.StatusOK, entry.MF2())
}

func (s *Server) micropubConfig(w http.ResponseWriter, r *http.Request) {
	syndications := []map[string]string{}
	for _, s := range s.syndicator.Config() {
		syndications = append(syndications, map[string]string{
			"uid":  s.UID,
			"name": s.Name,
		})
	}

	sections := []map[string]string{}
	for _, s := range s.c.Site.Sections {
		sections = append(sections, map[string]string{
			"uid":  s,
			"name": s,
		})
	}

	config := map[string]interface{}{
		"syndicate-to":   syndications,
		"channels":       sections,
		"media-endpoint": s.c.Server.AbsoluteURL("/micropub/media"),
	}

	if len(s.c.Micropub.PostTypes) > 0 {
		config["post-types"] = s.c.Micropub.PostTypes
	}

	s.serveJSON(w, http.StatusOK, config)
}

func (s *Server) micropubPost(w http.ResponseWriter, r *http.Request) {
	mr, err := micropub.ParseRequest(r)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var code int

	s.log.Infow("micropub: post", "request", mr)

	switch mr.Action {
	case micropub.ActionCreate:
		if !s.checkScope(w, r, "create") {
			return
		}
		code, err = s.micropubCreate(w, r, mr)
	case micropub.ActionUpdate:
		if !s.checkScope(w, r, "update") {
			return
		}
		code, err = s.micropubUpdate(w, r, mr)
	case micropub.ActionDelete:
		if !s.checkScope(w, r, "delete") {
			return
		}
		code, err = s.micropubDelete(w, r, mr)
	case micropub.ActionUndelete:
		if !s.checkScope(w, r, "undelete") {
			return
		}
		code, err = s.micropubUndelete(w, r, mr)
	default:
		code, err = http.StatusBadRequest, errors.New("invalid action")
	}

	if code >= 200 && code < 400 {
		w.WriteHeader(code)
	} else if code >= 400 && code < 500 {
		s.serveErrorJSON(w, code, "invalid_request", err.Error())
	} else if code >= 500 {
		s.log.Errorf("micropub: error on post: %s", err)
		s.serveErrorJSON(w, code, "server_error", err.Error())
	} else if err != nil {
		s.n.Error(fmt.Errorf("micropub: %w", err))
	}
}

func (s *Server) micropubCreate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	cmds := mf2.NewFlatHelper(mf2.Flatten(mr.Commands))
	slug := ""
	if s := cmds.String("mp-slug"); s != "" {
		slug = s
	}

	e, err := s.parser.FromMF2(mr.Properties, slug)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if s := cmds.Strings("mp-channel"); len(s) > 0 {
		e.Sections = append(e.Sections, s...)
	}

	if err := s.preSaveEntry(nil, e); err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.fs.SaveEntry(e)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var syndicators []string
	if s := cmds.Strings("mp-syndicate-to"); len(s) > 0 {
		syndicators = s
	}

	go s.postSaveEntry(nil, e, syndicators)
	http.Redirect(w, r, s.c.Server.BaseURL+e.ID, http.StatusAccepted)
	return 0, nil
}

func (s *Server) micropubUpdate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	old, err := s.fs.GetEntry(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	e, err := s.fs.TransformEntry(id, func(e *eagle.Entry) (*eagle.Entry, error) {
		mf := e.MF2()
		props := mf["properties"].(map[string][]interface{})
		newMf, err := micropub.Update(props, mr)
		if err != nil {
			return nil, err
		}

		err = e.Update(newMf)
		if err != nil {
			return nil, err
		}

		if err := s.preSaveEntry(old, e); err != nil {
			return nil, err
		}

		return e, nil
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go s.postSaveEntry(old, e, nil)
	http.Redirect(w, r, e.Permalink, http.StatusOK)
	return 0, nil
}

func (s *Server) micropubUndelete(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	old, err := s.fs.GetEntry(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	e, err := s.fs.TransformEntry(id, func(e *eagle.Entry) (*eagle.Entry, error) {
		e.Deleted = false
		if err := s.preSaveEntry(old, e); err != nil {
			return nil, err
		}
		return e, nil
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go s.postSaveEntry(old, e, nil)
	return http.StatusOK, nil
}

func (s *Server) micropubDelete(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	old, err := s.fs.GetEntry(id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	e, err := s.fs.TransformEntry(id, func(e *eagle.Entry) (*eagle.Entry, error) {
		e.Deleted = true

		if err := s.preSaveEntry(old, e); err != nil {
			return nil, err
		}

		return e, nil
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	go s.postSaveEntry(old, e, nil)
	return http.StatusOK, nil
}

func (s *Server) micropubParseURL(url string) (string, error) {
	if url == "" {
		return "", errors.New("url must be set")
	}

	if !strings.HasPrefix(url, s.c.Server.BaseURL) {
		return "", errors.New("invalid domain in url")
	}

	return strings.Replace(url, s.c.Server.BaseURL, "", 1), nil
}

func (s *Server) micropubMediaPost(w http.ResponseWriter, r *http.Request) {
	if !s.checkScope(w, r, "media") {
		return
	}

	err := r.ParseMultipartForm(20 << 20)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "file is too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", err.Error())
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
			s.serveErrorJSON(w, http.StatusBadRequest, "invalid_request", "request provides no file type")
			return
		}

		ext = mime.Extension()
	}

	location, err := s.media.UploadAnonymousMedia(ext, bytes.NewReader(raw))
	if err != nil {
		s.serveErrorJSON(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	http.Redirect(w, r, location, http.StatusCreated)
}

func (s *Server) checkScope(w http.ResponseWriter, r *http.Request, scope string) bool {
	scopes := s.getScopes(r)
	if !lo.Contains(scopes, scope) {
		s.serveErrorJSON(w, http.StatusForbidden, "insufficient_scope", "Insufficient scope.")
		return false
	}

	return true
}
