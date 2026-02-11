package server

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gabriel-vasile/mimetype"
	"github.com/samber/lo"
	"github.com/samber/lo/mutable"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/indielib/indieauth"
	"go.hacdias.com/indielib/micropub"
	"go.hacdias.com/maze"

	"github.com/go-playground/form/v4"
)

const (
	panelPath         = "/panel"
	panelBrowsePath   = panelPath + "/browse"
	panelEditPath     = panelPath + "/edit"
	panelNewPath      = panelPath + "/new"
	panelMentionsPtah = panelPath + "/mentions"
	panelTokensPath   = panelPath + "/tokens"
	panelCachePath    = panelPath + "/cache"
)

func (s *Server) servePanel(w http.ResponseWriter, r *http.Request, data *panelPage) {
	data.Title = "Panel"
	data.Actions = s.getActions()
	data.Success = r.URL.Query().Get("success")
	s.panelTemplate(w, r, http.StatusOK, panelTemplate, data)
}

func (s *Server) getSyndicators() []Syndicator {
	syndicators := []Syndicator{}
	for _, syndicator := range s.syndicators {
		syndicators = append(syndicators, syndicator.Syndicator())
	}

	return syndicators
}

type panelPage struct {
	Title         string
	Actions       []string
	Success       string
	MediaLocation string
	MediaPhoto    *core.Photo
}

func (s *Server) panelGet(w http.ResponseWriter, r *http.Request) {
	s.servePanel(w, r, &panelPage{})
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
	} else if err := r.ParseMultipartForm(20 << 20); err == nil {
		s.panelPostUpload(w, r)
		return
	}

	s.panelGet(w, r)
}

func (s *Server) panelPostAction(w http.ResponseWriter, r *http.Request) {
	actions := r.Form["action"]

	var err error
	for _, actionName := range actions {
		if fn, ok := s.actions[actionName]; ok {
			err = errors.Join(err, fn())
		}
	}
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	go s.build(false)
	http.Redirect(w, r, r.URL.Path+"?success=action", http.StatusSeeOther)
}

func (s *Server) panelPostUpload(w http.ResponseWriter, r *http.Request) {
	file, filename, ext, err := parseMediaRequest(w, r)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	mediaLocation, mediaPhoto, err := s.media.UploadMedia(filename, ext, bytes.NewReader(file))
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.servePanel(w, r, &panelPage{
		MediaLocation: mediaLocation,
		MediaPhoto:    mediaPhoto,
	})
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

	// Bonus: reverse the order in the posts directory to have the latest at the top.
	if strings.HasPrefix(filename, path.Join("/", core.ContentDirectory, core.PostsSection)) {
		mutable.Reverse(infos)
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

type editPage struct {
	Title       string
	Success     bool
	New         bool
	Path        string
	Content     string
	IsEntry     bool
	Syndicators []Syndicator
}

func (s *Server) panelEditGet(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Clean(strings.TrimPrefix(r.URL.Path, panelEditPath))

	info, err := s.core.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			s.panelTemplate(w, r, http.StatusOK, panelEditorTemplate, &editPage{
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

	pageData := &editPage{
		Title:   "Editor",
		Success: r.URL.Query().Get("success") == "true",
		Path:    filename,
		Content: data,
	}

	if e, err := s.core.GetEntryByFilename(filename); err == nil && e.IsPost() {
		pageData.IsEntry = true
		pageData.Syndicators = lo.Map(s.getSyndicators(), func(s Syndicator, _ int) Syndicator {
			s.Default = false
			return s
		})
	} else if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, core.ErrIgnoredEntry) {
		s.panelError(w, r, http.StatusBadRequest, fmt.Errorf("error getting entry by filename: %w", err))
		return
	}

	s.panelTemplate(w, r, http.StatusOK, panelEditorTemplate, pageData)
}

type editRequest struct {
	Content           string   `form:"content"`
	Syndicators       []string `form:"syndicators"`
	SyndicationStatus string   `form:"syndication-status"`
}

func (s *Server) panelEditPost(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Clean(strings.TrimPrefix(r.URL.Path, panelEditPath))

	err := r.ParseForm()
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	var req editRequest

	decoder := form.NewDecoder()
	err = decoder.Decode(&req, r.Form)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	req.Content = string(normalizeLineEndings([]byte(req.Content)))

	if oldEntry, err := s.core.GetEntryByFilename(filename); err == nil && oldEntry.IsPost() {
		previousLinks, _ := s.core.GetEntryLinks(oldEntry, true)

		e, err := s.core.GetEntryFromContent(oldEntry.ID, req.Content)
		if err != nil {
			s.panelError(w, r, http.StatusBadRequest, err)
			return
		}

		err = s.saveEntryWithHooks(e, postSaveEntryOptions{
			syndicators:       req.Syndicators,
			syndicationStatus: req.SyndicationStatus,
			previousLinks:     previousLinks,
		})
		if err != nil {
			s.panelError(w, r, http.StatusInternalServerError, err)
			return
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, core.ErrIgnoredEntry) {
		s.panelError(w, r, http.StatusBadRequest, fmt.Errorf("error getting entry by filename: %w", err))
		return
	}

	err = s.core.WriteFile(filename, []byte(req.Content), "editor: update "+filename)
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, r.URL.Path+"?success=true", http.StatusSeeOther)
}

type newPage struct {
	Title       string
	Categories  []micropub.Channel
	Syndicators []Syndicator
}

func (s *Server) panelNewGet(w http.ResponseWriter, r *http.Request) {
	s.panelTemplate(w, r, http.StatusOK, panelNewTemplate, &newPage{
		Title:       "New",
		Syndicators: s.getSyndicators(),
		Categories: []micropub.Channel{
			// TODO: these could perhaps be defined in the config. Then, we could
			// also set one as "long-form" and use that value in plugins/atproto
			// for syndication instead of hardcoding.
			{UID: "writings", Name: "Writings"},
			{UID: "photos", Name: "Photos"},
		},
	})
}

type newRequest struct {
	Title    string   `form:"title"`
	Slug     string   `form:"slug"`
	Content  string   `form:"content"`
	Category string   `form:"category"`
	Tags     []string `form:"tags"`
	Location string   `form:"location"`
	Photos   []struct {
		URL   string `form:"url"`
		Title string `form:"title"`
	} `form:"photos"`
	Syndicators       []string `form:"syndicators"`
	SyndicationStatus string   `form:"syndication-status"`
}

func (s *Server) panelNewPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	var req newRequest

	decoder := form.NewDecoder()
	err = decoder.Decode(&req, r.Form)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	if req.Title == "" || req.Slug == "" || req.Content == "" || req.Category == "" {
		s.panelError(w, r, http.StatusBadRequest, errors.New("title, slug, content and category are required"))
		return
	}

	id := core.NewPostID(req.Slug, time.Now())
	var e *core.Entry

	if strings.HasPrefix(req.Content, "---") {
		e, err = s.core.GetEntryFromContent(id, req.Content)
		if err != nil {
			s.panelError(w, r, http.StatusBadRequest, err)
			return
		}
	} else {
		e = s.core.NewBlankEntry(id)
		e.Content = req.Content
	}

	e.Title = req.Title
	e.Categories = []string{req.Category}
	e.Tags = req.Tags

	if len(req.Photos) > 0 {
		if len(e.Photos) != 0 {
			s.panelError(w, r, http.StatusBadRequest, errors.New("cannot specify photos in form when entry already has photos in content"))
			return
		}

		for _, p := range req.Photos {
			e.Photos = append(e.Photos, core.Photo{
				URL:   p.URL,
				Title: p.Title,
			})
		}
	}

	if req.Location != "" {
		parsedLocation, err := maze.ParseLocation(req.Location)
		if err != nil {
			s.panelError(w, r, http.StatusBadRequest, err)
			return
		}

		e.Location = parsedLocation
	}

	err = s.updateEntryWithPhotos(e)
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	err = s.saveEntryWithHooks(e, postSaveEntryOptions{
		syndicators:       req.Syndicators,
		syndicationStatus: req.SyndicationStatus,
	})
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, e.Permalink, http.StatusSeeOther)
}

func (s *Server) updateEntryWithPhotos(e *core.Entry) error {
	// Define prefix for the photos that will be uploaded
	parts := strings.Split(strings.TrimSuffix(e.ID, "/"), "/")
	slug := parts[len(parts)-1]
	prefix := fmt.Sprintf("%04d-%02d-%02d-%s", e.Date.Year(), e.Date.Month(), e.Date.Day(), slug)

	for i := range e.Photos {
		url := e.Photos[i].URL

		if strings.HasPrefix(url, "cache:") {
			data, ok := s.mediaCache.GetIfPresent(url)
			if !ok {
				return fmt.Errorf("photo %q not found in cache", url)
			}

			filename := prefix
			if len(e.Photos) > 1 {
				filename += fmt.Sprintf("-%02d", i+1)
			}

			ext := filepath.Ext(url)
			location, photo, err := s.media.UploadMedia(filename, ext, bytes.NewBuffer(data))
			if err != nil {
				return fmt.Errorf("failed to upload photo: %w", err)
			}

			if photo != nil {
				e.Photos[i].URL = photo.URL
				e.Photos[i].Width = photo.Width
				e.Photos[i].Height = photo.Height
			} else {
				e.Photos[i].URL = location
			}

			s.mediaCache.Invalidate(url)
		}
	}

	return nil
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

func (s *Server) panelCachePost(w http.ResponseWriter, r *http.Request) {
	file, _, ext, err := parseMediaRequest(w, r)
	if err != nil {
		s.panelError(w, r, http.StatusBadRequest, err)
		return
	}

	filename := fmt.Sprintf("cache:%x%s", sha256.Sum256(file), ext)
	s.mediaCache.Set(filename, file)
	_, _ = w.Write([]byte(filename))
}

func normalizeLineEndings(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.ReplaceAll(d, []byte{13, 10}, []byte{10})
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.ReplaceAll(d, []byte{13}, []byte{10})
	return d
}

func parseMediaRequest(w http.ResponseWriter, r *http.Request) ([]byte, string, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)

	err := r.ParseMultipartForm(20 << 20)
	if err != nil {
		return nil, "", "", err
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, "", "", err
	}
	defer func() {
		_ = file.Close()
	}()

	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, "", "", err
	}

	ext := filepath.Ext(header.Filename)
	filename := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
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
			return nil, "", "", errors.New("cannot deduce mimetype")
		}

		ext = mime.Extension()
	}

	return raw, filename, ext, nil
}
