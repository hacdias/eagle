package server

import (
	"net/http"
	"path"
	"strings"
)

type notFoundRedirectRespWr struct {
	http.ResponseWriter // We embed http.ResponseWriter
	status              int
}

func (w *notFoundRedirectRespWr) WriteHeader(status int) {
	w.status = status // Store the status for our own use
	if status != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *notFoundRedirectRespWr) Write(p []byte) (int, error) {
	if w.status != http.StatusNotFound {
		return w.ResponseWriter.Write(p)
	}
	return len(p), nil // Lie that we successfully written it
}

func staticHandler(dir string) func(http.ResponseWriter, *http.Request) {
	fs := http.FileServer(http.Dir(dir))

	return func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "application/activity+json") || strings.Contains(accept, `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`) {
			// TODO: do things
		}

		nfw := &notFoundRedirectRespWr{ResponseWriter: w}
		fs.ServeHTTP(nfw, r)

		if nfw.status == http.StatusNotFound {
			w.Header().Del("Content-Type") // Let http.ServeFile set the correct header
			http.ServeFile(w, r, path.Join(dir, "404.html"))
		}
	}
}
