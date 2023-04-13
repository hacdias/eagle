package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/pkg/contenttype"
)

const (
	templatesExtension string = ".html"
	templatesDirectory string = "eagle/templates"

	// templateSearch    string = "search"
	templateError     string = "error"
	templateLogin     string = "login"
	templateAuth      string = "auth"
	templateNew       string = "new"
	templateEdit      string = "edit"
	templateDashboard string = "dashboard"
	templateAdminBar  string = "admin-bar"

	eagleTemplatePath     string = "/eagle-template*"
	eagleTemplateFilePath string = "eagle-template/index.html"
)

func (s *Server) loadTemplates() error {
	parsed := map[string]*template.Template{}

	err := s.fs.Walk(templatesDirectory, func(filename string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		basename := filepath.Base(info.Name())
		ext := filepath.Ext(basename)

		id := strings.TrimPrefix(filename, templatesDirectory)
		id = strings.TrimSuffix(id, ext)
		id = strings.TrimSuffix(id, "/")
		id = strings.TrimPrefix(id, "/")

		if ext != templatesExtension {
			return nil
		}

		raw, err := s.fs.ReadFile(filename)
		if err != nil {
			return err
		}

		parsed[id], err = template.New(id).Parse(string(raw))
		return err
	})

	if err != nil {
		return err
	}

	s.templates = parsed
	return nil
}

// captureResponseWriter captures the content of an HTML response. If the response
// is HTML, the Content-Length header will also be removed. All other headers,
// including status, will be sent.
type captureResponseWriter struct {
	http.ResponseWriter
	captured bool
	body     []byte
}

func (w *captureResponseWriter) WriteHeader(status int) {
	contentType := w.Header().Get("Content-Type")

	if strings.Contains(contentType, contenttype.HTML) {
		// Delete Content-Length header to avoid browser issues. We could've added
		// the size of the rendered admin bar and then re-set the header. However,
		// I saw no practical benefit on doing so.
		w.Header().Del("Content-Length")
		w.captured = true
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *captureResponseWriter) Write(p []byte) (int, error) {
	if w.captured {
		w.body = append(w.body, p...)
		return len(p), nil
	}

	return w.ResponseWriter.Write(p)
}

func (s *Server) withAdminBar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// NOTE: this is a very basic attempt at detecting whether or not this
		// is an HTML request. Depending on this, we set the cache control headers.
		ext := path.Ext(r.URL.Path)
		isHTML := ext == "" || ext == ".html"
		setCacheControl(w, isHTML)

		if s.isLoggedIn(r) && isHTML {
			// Ensure that logged in requests to HTML files are not cached by the browser.
			delEtagHeaders(r)

			// Potentially capture request.
			crw := &captureResponseWriter{ResponseWriter: w}
			next.ServeHTTP(crw, r)

			// Not capture, move on.
			if !crw.captured {
				return
			}

			html, err := s.renderAdminBar(r.URL.Path)
			if err == nil {
				tag := []byte("<body>")
				html = append([]byte("<body>"), html...)
				_, err = w.Write(bytes.Replace(crw.body, tag, html, 1))
			}

			if err != nil {
				s.log.Warn("could not inject admin bar", err)
			}

			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) renderAdminBar(path string) ([]byte, error) {
	tpl, ok := s.templates[templateAdminBar]
	if !ok {
		return nil, fmt.Errorf("template %s not found", templateAdminBar)
	}

	var buf bytes.Buffer
	err := tpl.Execute(&buf, map[string]interface{}{
		"ID": path,
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type renderData struct {
	Title    string
	LoggedIn bool
	Data     interface{}
}

func (s *Server) serveHTML(w http.ResponseWriter, r *http.Request, data *renderData, template string, statusCode int) {
	data.LoggedIn = s.isLoggedIn(r)

	raw, err := s.staticFs.ReadFile(eagleTemplateFilePath)
	if err != nil {
		s.n.Error(fmt.Errorf("%s file not found in public directory", eagleTemplateFilePath))
		return
	}

	tpl, ok := s.templates[template]
	if !ok {
		s.n.Error(fmt.Errorf("serving html for %s: template %s not found", r.URL.Path, template))
		return
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, data)
	if err != nil {
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
	}

	v := string(raw)
	v = strings.Replace(v, "#BODY", buf.String(), -1)
	v = strings.Replace(v, "#TITLE", data.Title, -1)

	w.Header().Set("Content-Type", contenttype.HTMLUTF8)
	w.WriteHeader(statusCode)
	w.Write([]byte(v))
}

func (s *Server) serveErrorHTML(w http.ResponseWriter, r *http.Request, code int, err error) {
	if err != nil {
		s.log.Error(err)
	}

	w.Header().Del("Cache-Control")

	data := map[string]interface{}{
		"Code": code,
	}

	if err != nil {
		data["Error"] = err.Error()
	}

	rd := &renderData{
		Title: fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Data:  data,
	}

	s.serveHTML(w, r, rd, templateError, code)
}

func (s *Server) serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", contenttype.JSONUTF8)
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.n.Error(fmt.Errorf("serving html: %w", err))
	}
}

func (s *Server) serveErrorJSON(w http.ResponseWriter, code int, err, errDescription string) {
	s.serveJSON(w, code, map[string]string{
		"error":             err,
		"error_description": errDescription,
	})
}
