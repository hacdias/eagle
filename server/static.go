package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/hacdias/eagle/config"
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

func staticHandler(c *config.Config) func(http.ResponseWriter, *http.Request) {
	httpdir := http.Dir(c.Hugo.Destination)
	fs := http.FileServer(httpdir)

	domain, err := url.Parse(c.Domain)
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
		json.NewEncoder(w).Encode(data)
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
			http.ServeFile(w, r, path.Join(c.Hugo.Destination, "404.html"))
		}
	}
}

// TODO(activitypub)
// 	- GOAL: support MF2, JF2 and AS2
//	- PRIORITY: support AS2
//	- I can convert MF2 to JF2 quite easily
//	- AS2: generate with Hugo or programatically?
//	- Handle special AS2 for homepage or /hacdias (pick one)
//	- Handle webfinger
//	- SOLUTIONS
//			1. Generate on the fly.
//			2. Generate on the fly and cache. Evict when there are updates.
//			2. Run script after 'hugo build' that converts the HTML to MF2, JF2 and AS2

/*
func mf2ToAs2(data *microformats.Data) *as2 {
	if len(data.Items) < 1 {
		return nil
	}

	a := &as2{
		Context: []string{
			"https://www.w3.org/ns/activitystreams",
		},
		To: []string{
			"https://www.w3.org/ns/activitystreams#Public",
		},
		MediaType: "text/html",
		// ID:        data.Items[0].
		// ID:        data.Items[0].Properties["url"],
	}

	return a
}

type as2 struct {
	Context      []string `json:"@context,omitempty"`
	To           []string `json:"to,omitempty"`
	Published    string   `json:"published,omitempty"`
	Updated      string   `json:"updated,omitempty"`
	ID           string   `json:"id,omitempty"`
	URL          string   `json:"url,omitempty"`
	Content      string   `json:"content,omitempty"`
	MediaType    string   `json:"mediaType,omitempty"`
	Name         string   `json:"name,omitempty"`
	Type         string   `json:"type,omitempty"`
	AttributedTo string   `json:"attributedTo,omitempty"`
	InReplyTo    string   `json:"inReplyTo,omitempty"`
}

/*
{
  "published": {{ dateFormat "2006-01-02T15:04:05-07:00" .Date | jsonify }},
  "updated": {{ dateFormat "2006-01-02T15:04:05-07:00" .Lastmod | jsonify }},
  "id": "{{ .Permalink }}",
  "url": "{{ .Permalink }}",
  "content": {{ partialCached "cleaned-content.html" . .Permalink | jsonify }},
  {{ with .Title }}"name": {{ . | jsonify }},{{ end }}
  "type": "{{ if eq .Section "articles" }}Article{{ else }}Note{{ end }}",
  "attributedTo": "{{ "" | absLangURL }}"{{ if .Params.properties }}{{ if isset .Params.properties "in-reply-to" }},
  "inReplyTo": "{{ index .Params.properties "in-reply-to" }}"
  {{ end }}{{ end }}
}
*/
