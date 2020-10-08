package server

import (
	"io"
	"net/http"
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

func staticHandler(dir string) func(http.ResponseWriter, *http.Request) {
	httpdir := http.Dir(dir)
	fs := http.FileServer(httpdir)

	findActivity := func(url string) string {
		filepath := path.Join(url, "index.as2")
		// TODO: I'd prefer just stat and not open. Is it better like this
		// or serve it directly?
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

	return func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		acceptsHTML := strings.Contains(accept, "text/html")
		acceptsActivity := strings.Contains(accept, "application/activity+json") || strings.Contains(accept, `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)
		acceptsMf2 := strings.Contains(accept, "application/mf2+json")

		if strings.HasSuffix(r.URL.Path, ".as2") {
			w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
		}

		if !acceptsHTML {
			if acceptsActivity {
				if as2 := findActivity(r.URL.Path); as2 != "" {
					w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
					r.URL.Path = as2
				}
			}

			if acceptsMf2 {
				// TODO: simplify and add alternate meta on HTML
				fd := getHTML(r.URL.Path)
				if fd != nil {
					defer fd.Close()
					data := microformats.Parse(fd, r.URL)
					if data != nil {
						serveJSON(w, http.StatusOK, data)
						return
					}
				}
			}
		}

		nfw := &notFoundRedirectRespWr{ResponseWriter: w}
		fs.ServeHTTP(nfw, r)

		if nfw.status == http.StatusNotFound {
			w.Header().Del("Content-Type") // Let http.ServeFile set the correct header
			http.ServeFile(w, r, path.Join(dir, "404.html"))
		}
	}
}
