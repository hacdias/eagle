package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"willnorris.com/go/microformats"
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

func (s *Server) staticHandler() http.HandlerFunc {
	httpdir := http.Dir(s.c.Hugo.Destination)
	fs := http.FileServer(httpdir)

	domain, err := url.Parse(s.c.Domain)
	if err != nil {
		panic("domain is invalid")
	}

	findActivity := func(url string) string {
		filepath := path.Join(url, "index.as2")
		if fd, err := httpdir.Open(filepath); err == nil {
			fd.Close()
			return filepath
		}

		return ""
	}

	getHTML := func(url string) io.ReadCloser {
		if !strings.HasSuffix(url, ".html") {
			url = path.Join(url, "index.html")
		}

		fd, err := httpdir.Open(url)
		if err != nil {
			return nil
		}

		return fd
	}

	getMf2 := func(url *url.URL) *microformats.Data {
		fd := getHTML(url.Path)
		if fd == nil {
			return nil
		}
		defer fd.Close()

		return microformats.Parse(fd, url)
	}

	tryActivity := func(w http.ResponseWriter, r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, ".as2") {
			return false
		}
		as2 := findActivity(r.URL.Path)
		if as2 == "" {
			return false
		}

		w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
		r.URL.Path = as2
		return false
	}

	tryMf2 := func(w http.ResponseWriter, r *http.Request) bool {
		actualURL := r.URL
		if strings.HasSuffix(actualURL.Path, "index.mf2") {
			actualURL.Path = strings.TrimSuffix(actualURL.Path, "index.mf2")
		}

		data := getMf2(actualURL)
		if data == nil {
			return false
		}

		w.Header().Set("Content-Type", "application/mf2+json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			s.Errorf("error while serving json: %s", err)
		}
		return true
	}

	return func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		acceptsHTML := strings.Contains(accept, "text/html")
		acceptsActivity := strings.Contains(accept, "application/activity+json")
		acceptsMf2 := strings.Contains(accept, "application/mf2+json")

		r.URL.Scheme = domain.Scheme
		r.URL.Host = domain.Host

		if strings.HasSuffix(r.URL.Path, "index.as2") || (!acceptsHTML && acceptsActivity) {
			_ = tryActivity(w, r)
		}

		if strings.HasSuffix(r.URL.Path, "index.mf2") || (!acceptsHTML && acceptsMf2) {
			if ok := tryMf2(w, r); ok {
				return
			}
		}

		nfw := &notFoundRedirectRespWr{ResponseWriter: w}
		fs.ServeHTTP(nfw, r)

		if nfw.status == http.StatusNotFound {
			w.Header().Del("Content-Type") // Let http.ServeFile set the correct header
			http.ServeFile(w, r, path.Join(s.c.Hugo.Destination, "404.html"))
		}
	}
}
