package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/gabriel-vasile/mimetype"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/indielib/indieauth"
)

const (
	panelPath         = "/panel"
	panelMentionsPtah = panelPath + "/mentions"
	panelTokensPath   = panelPath + "/tokens"
	panelBrowsePath   = panelPath + "/browse"
	panelEditPath     = panelPath + "/edit"
)

type panelPage struct {
	Title              string
	Actions            []string
	ActionSuccess      bool
	Token              string
	MediaLocation      string
	MediaPhoto         *core.Photo
	WebmentionsSuccess bool
}

func (s *Server) panelGet(w http.ResponseWriter, r *http.Request) {
	s.servePanel(w, r, &panelPage{})
}

type browserPage struct {
	Title string
	Path  string
	Files []fs.FileInfo
}

func (s *Server) panelBrowserGet(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Clean(strings.TrimPrefix(r.URL.Path, panelBrowsePath))

	info, err := s.core.Stat(filename)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	if !info.IsDir() {
		http.Redirect(w, r, panelEditPath+filename, http.StatusTemporaryRedirect)
		return
	}

	canonical := path.Join(panelBrowsePath, filename) + "/"
	if r.URL.Path != canonical {
		http.Redirect(w, r, canonical, http.StatusTemporaryRedirect)
		return
	}

	infos, err := s.core.ReadDir(filename)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	s.panelTemplate(w, r, http.StatusOK, panelBrowserTemplate, &browserPage{
		Title: "Browser",
		Path:  filename,
		Files: infos,
	})
}

func (s *Server) panelBrowserPost(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Clean(strings.TrimPrefix(r.URL.Path, panelBrowsePath))

	err := r.ParseForm()
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	if fname := r.FormValue("filename"); fname != "" {
		http.Redirect(w, r, panelEditPath+path.Join(filename, fname), http.StatusSeeOther)
		return
	}

	dirname := r.FormValue("dirname")
	if dirname == "" {
		s.panelError(w, r, http.StatusBadRequest, errors.New("dirname cannot be empty"))
		return
	}

	filename = filepath.Join(filename, dirname)
	err = s.core.MkdirAll(filename)
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, panelBrowsePath+path.Clean(filename), http.StatusSeeOther)
}

type editorPage struct {
	Title   string
	Success bool
	New     bool
	Path    string
	Content string
}

func (s *Server) panelEditGet(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Clean(strings.TrimPrefix(r.URL.Path, panelEditPath))

	info, err := s.core.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			s.panelTemplate(w, r, http.StatusOK, panelEditorTemplate, &editorPage{
				Title:   "Editor",
				New:     true,
				Path:    filename,
				Content: "",
			})
			return
		}

		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	if info.IsDir() {
		http.Redirect(w, r, panelBrowsePath+filename, http.StatusTemporaryRedirect)
		return
	}

	canonical := path.Join(panelEditPath, filename)
	if r.URL.Path != canonical {
		http.Redirect(w, r, canonical, http.StatusTemporaryRedirect)
		return
	}

	val, err := s.core.ReadFile(filename)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	data := string(val)
	if !utf8.ValidString(data) {
		s.panelError(w, r, http.StatusPreconditionFailed, errors.New("file is not text file"))
		return
	}

	s.panelTemplate(w, r, http.StatusOK, panelEditorTemplate, &editorPage{
		Title:   "Editor",
		Success: r.URL.Query().Get("success") == "true",
		Path:    filename,
		Content: data,
	})
}

func (s *Server) panelEditPost(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Clean(strings.TrimPrefix(r.URL.Path, panelEditPath))

	err := r.ParseForm()
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	content := []byte(r.FormValue("content"))
	content = normalizeLineEndings(content)

	err = s.core.WriteFile(filename, content, "editor: update "+filename)
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, r.URL.Path+"?success=true", http.StatusSeeOther)
}

func (s *Server) panelPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	if r.Form.Get("action") != "" {
		s.panelPostAction(w, r)
		return
	} else if wm := r.Form.Get("webmention"); wm != "" {
		s.panelPostWebmention(w, r)
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
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.build(false)
	s.servePanel(w, r, data)
}

func (s *Server) panelPostWebmention(w http.ResponseWriter, r *http.Request) {
	permalink := r.Form.Get("webmention")
	err := s.core.SendWebmentions(permalink)
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.servePanel(w, r, &panelPage{WebmentionsSuccess: true})
}

func (s *Server) panelPostUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	raw, err := io.ReadAll(file)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
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
			s.panelError(w, r, http.StatusBadRequest, err)
			return
		}

		ext = mime.Extension()
	}

	mediaLocation, mediaPhoto, err := s.media.UploadMedia(strings.TrimSuffix(header.Filename, ext), ext, bytes.NewReader(raw))
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.servePanel(w, r, &panelPage{
		MediaLocation: mediaLocation,
		MediaPhoto:    mediaPhoto,
	})
}

func (s *Server) servePanel(w http.ResponseWriter, r *http.Request, data *panelPage) {
	data.Title = "Panel"
	data.Actions = s.getActions()
	s.panelTemplate(w, r, http.StatusOK, panelTemplate, data)
}

type mentionsPage struct {
	Title    string
	Mentions []*core.Mention
}

func (s *Server) panelMentionsGet(w http.ResponseWriter, r *http.Request) {
	mentions, err := s.bolt.GetMentions(r.Context())
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, fmt.Errorf("error getting mentions: %w", err))
		return
	}

	s.panelTemplate(w, r, http.StatusOK, panelMentionsTemplate, &mentionsPage{
		Title:    "Mentions",
		Mentions: mentions,
	})
}

func (s *Server) panelMentionsPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	action := r.Form.Get("action")
	id := r.Form.Get("id")

	switch action {
	case "approve":
		e, err := s.bolt.GetMention(r.Context(), id)
		if err != nil {
			s.panelError(w, r, http.StatusInternalServerError, err)
			return
		}

		if !e.Private {
			if err := s.core.AddOrUpdateWebmention(e.EntryID, e, ""); err != nil {
				s.panelError(w, r, http.StatusInternalServerError, fmt.Errorf("error adding or updating webmention: %w", err))
				return
			}

			go func() {
				_ = s.core.Build(false)
			}()
		}

		fallthrough
	case "delete":
		err := s.bolt.DeleteMention(r.Context(), id)
		if err != nil {
			s.panelError(w, r, http.StatusInternalServerError, err)
			return
		}
	default:
		s.panelError(w, r, http.StatusBadRequest, fmt.Errorf("invalid action: %s", action))
		return
	}

	http.Redirect(w, r, r.URL.Path, http.StatusFound)
}

type tokenPage struct {
	Title string
	Token string
}

func (s *Server) panelTokensGet(w http.ResponseWriter, r *http.Request) {
	s.panelTemplate(w, r, http.StatusOK, panelTokensTemplate, &tokenPage{
		Title: "Tokens",
	})
}

func (s *Server) panelTokensPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	data := &tokenPage{
		Title: "Tokens",
	}

	clientID := r.Form.Get("client_id")
	scope := r.Form.Get("scope")
	expiry, err := handleExpiry(r.Form.Get("expiry"))
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, fmt.Errorf("expiry param is invalid: %w", err))
		return
	}

	if err := indieauth.IsValidClientIdentifier(clientID); err != nil {
		s.panelError(w, r, http.StatusBadRequest, fmt.Errorf("invalid client_id: %w", err))
		return
	}

	signed, err := s.generateToken(clientID, scope, expiry)
	if err == nil {
		data.Token = signed
	} else {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.panelTemplate(w, r, http.StatusOK, panelTokensTemplate, data)
}

func normalizeLineEndings(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.ReplaceAll(d, []byte{13, 10}, []byte{10})
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.ReplaceAll(d, []byte{13}, []byte{10})
	return d
}
