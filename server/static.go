package server

import (
	"net/http"
)

func (s *Server) generalHandler(w http.ResponseWriter, r *http.Request) {
	if url, ok := s.redirects[r.URL.Path]; ok {
		http.Redirect(w, r, url, http.StatusMovedPermanently)
		return
	}

	if _, ok := s.gone[r.URL.Path]; ok {
		s.serveErrorHTML(w, r, http.StatusGone, nil)
		return
	}

	nfw := &notFoundResponseWriter{ResponseWriter: w}
	s.staticFs.ServeHTTP(nfw, r)

	if nfw.status == http.StatusNotFound {
		e, err := s.core.GetEntry(r.URL.Path)
		if err == nil && e.Deleted() {
			s.serveErrorHTML(w, r, http.StatusGone, nil)
		} else {
			s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		}
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
