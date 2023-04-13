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

	"github.com/PuerkitoBio/goquery"
	"github.com/hacdias/eagle/pkg/contenttype"
)

const (
	templatesExtension string = ".html"
	templatesDirectory string = "eagle/templates"

	templateAdminBar string = "admin-bar"
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

func (s *Server) renderTemplate(template string, data interface{}) ([]byte, error) {
	var buf bytes.Buffer

	tpl, ok := s.templates[template]
	if !ok {
		return nil, fmt.Errorf("template %s not found", template)
	}

	err := tpl.Execute(&buf, data)
	if err != nil {
		return nil, fmt.Errorf("error executing template %s: %w", template, err)
	}

	return buf.Bytes(), nil
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

			e, _ := s.fs.GetEntry(r.URL.Path)

			html, err := s.renderTemplate(templateAdminBar, map[string]interface{}{
				"Entry": e,
			})
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

func (s *Server) serveErrorHTML(w http.ResponseWriter, r *http.Request, code int, reqErr error) {
	if reqErr != nil {
		s.log.Error(reqErr)
	}

	w.Header().Del("Cache-Control")
	f, err := s.staticFs.ReadFile("404.html")
	if err != nil {
		s.log.Error(err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(f))
	if err != nil {
		s.log.Error(err)
		return
	}

	errorTitle := fmt.Sprintf("%d %s", code, http.StatusText(code))

	doc.Find("title").SetText(errorTitle + " - " + s.c.Site.Title)
	doc.Find("eagle-error-title").ReplaceWithHtml(errorTitle)
	selector := fmt.Sprintf("eagle-error[error~=\"%d\"]", code)
	errorNode := doc.Find(selector)
	errorNode.ReplaceWithSelection(errorNode.Children())
	doc.Find("eagle-error").Remove()

	detailsNode := doc.Find("eagle-error-details")
	if reqErr != nil && (s.isLoggedIn(r) || code < 500) {
		detailsNode.Find("eagle-error-details-value").ReplaceWithHtml(reqErr.Error())
		detailsNode.ReplaceWithSelection(detailsNode.Children())
	} else {
		detailsNode.Remove()
	}

	s.serveDocument(w, r, doc, code)
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

func (s *Server) getTemplateDocument(path string) (*goquery.Document, error) {
	f, err := s.staticFs.ReadFile(filepath.Join(path, "index.html"))
	if err != nil {
		return nil, err
	}

	return goquery.NewDocumentFromReader(bytes.NewReader(f))
}

func (s *Server) serveDocument(w http.ResponseWriter, r *http.Request, doc *goquery.Document, statusCode int) {
	doc.Find("no-eagle-page").Remove()
	pageNode := doc.Find("eagle-page")
	pageNode.ReplaceWithSelection(pageNode.Children())

	html, err := doc.Html()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
		return
	}

	w.Header().Set("Content-Type", contenttype.HTMLUTF8)
	w.WriteHeader(statusCode)
	_, err = w.Write([]byte(html))
	if err != nil {
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
	}
}
