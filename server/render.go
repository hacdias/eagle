package server

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
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
)

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
