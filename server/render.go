package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	templateAdminBar string = "admin-bar.html"
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

	if strings.Contains(contentType, "text/html") {
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

			doc, err := goquery.NewDocumentFromReader(bytes.NewReader(crw.body))
			if err != nil {
				s.log.Warn("could not parse document", err)
				return
			}

			var buf bytes.Buffer
			err = s.templates.ExecuteTemplate(&buf, templateAdminBar, nil)
			if err == nil {
				doc.Find("body").PrependHtml(buf.String())
				raw, err := doc.Html()
				if err != nil {
					s.log.Warn("could not convert document", err)
					return
				}

				_, err = w.Write([]byte(raw))
				if err != nil {
					s.log.Warn("could not write document", err)
					return
				}
			} else {
				s.log.Warn("could not inject admin bar", err)
			}

			return
		}

		next.ServeHTTP(w, r)
	})
}

type errorPage struct {
	Status  int
	Message string
}

func (s *Server) serveErrorHTML(w http.ResponseWriter, r *http.Request, code int, reqErr error) {
	if reqErr != nil {
		s.log.Error(reqErr)
	}

	w.Header().Del("Cache-Control")

	data := &errorPage{
		Status: code,
	}

	if reqErr != nil && (s.isLoggedIn(r) || code < 500) {
		data.Message = reqErr.Error()
	}

	s.renderTemplateWithContent(w, r, "error.html", &pageData{
		Title: fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Data:  data,
	})
}

func (s *Server) serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

type pageData struct {
	Title string
	Data  interface{}
}

func (s *Server) renderTemplateWithContent(w http.ResponseWriter, r *http.Request, template string, p *pageData) {
	fd, err := s.staticFs.ReadFile(filepath.Join("/eagle/", "index.html"))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(fd))
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	var buf bytes.Buffer
	err = s.templates.ExecuteTemplate(&buf, template, p)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	doc.Find("title").SetText(strings.Replace(doc.Find("title").Text(), "Eagle", p.Title, 1))

	pageNode := doc.Find("eagle-page")
	pageNode.ReplaceWithHtml(buf.String())

	html, err := doc.Html()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(html))
	if err != nil {
		s.n.Error(fmt.Errorf("serving html for %s: %w", r.URL.Path, err))
	}
}
