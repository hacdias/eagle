package server

import (
	"net/http"
	"path"
	"strconv"
	"strings"
)

func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	// NOTE: previously we'd do a staticFs read lock here. However, removing
	// it increased performance dramatically. Hopefully there's no consequences.

	// TODO: somehow improve the detection of whether or not the page is HTML.
	// We cannot do it on a response writer wrapper because we need to know before
	// it reaches the http.FileServer.
	ext := path.Ext(r.URL.Path)
	isHTML := ext == "" || ext == ".html"
	setCacheHeaders(w, isHTML)

	if isAuthd, ok := r.Context().Value(&authContextKey).(bool); ok && isAuthd {
		if isHTML {
			// Ensure that authenticated requests to HTML files do not trigger
			// a Not Modified responnse from http.FileServer.
			delEtagHeaders(r)
		}

		w = &adminBarResponseWriter{
			ResponseWriter: w,
			s:              s,
			p:              r.URL.Path,
		}
	}

	nfw := &notFoundResponseWriter{ResponseWriter: w}
	s.staticFs.ServeHTTP(nfw, r)

	if nfw.status == http.StatusNotFound {
		bytes, err := s.staticFs.ReadFile("404.html")
		if err != nil {
			s.serveError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Del("Cache-Control")

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(bytes)
	}
}

// notFoundResponseWriter wraps a Response Writer to capture 404 requests.
// In case it is a 404 request, then we do not write the body.
type notFoundResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *notFoundResponseWriter) WriteHeader(status int) {
	w.status = status
	if status != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *notFoundResponseWriter) Write(p []byte) (int, error) {
	if w.status != http.StatusNotFound {
		return w.ResponseWriter.Write(p)
	}
	// Lie that we successfully written it
	return len(p), nil
}

type adminBarResponseWriter struct {
	http.ResponseWriter
	s *Server
	p string
}

func (w *adminBarResponseWriter) WriteHeader(status int) {
	if status == http.StatusOK && strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		length, _ := strconv.Atoi(w.Header().Get("Content-Length"))
		html, err := w.s.renderAdminBar(w.p)
		if err == nil {
			length += len(html)
			w.Header().Set("Content-Length", strconv.Itoa(length))
			w.ResponseWriter.WriteHeader(status)
			_, err = w.Write(html)
			if err != nil {
				w.s.Warn("could not write admin bar", err)
			}
		} else {
			w.s.Warn("could not render admin bar", err)
		}

		return
	}

	w.ResponseWriter.WriteHeader(status)
}
