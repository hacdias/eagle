package server

import (
	"net/http"
	"strconv"
)

func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	// NOTE: previously we'd do a staticFs read lock here. However, removing
	// it increased performance dramatically. Hopefully there's no consequences.
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
