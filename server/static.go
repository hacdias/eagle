package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/spf13/afero"
	"willnorris.com/go/microformats"
)

const activityContentType = "application/activity+json"
const activityExt = ".as2"
const mf2ContentType = "application/mf2+json"
const mf2Ext = ".mf2"

// TODO: add jf2
// const jf2ContentType = "application/jf2+json"
// const jf2Ext = ".jf2"

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

func (s *Server) getHTML(url string) io.ReadCloser {
	if !strings.HasSuffix(url, ".html") {
		url = path.Join(url, "index.html")
	}

	fd, err := s.fs.Open(url)
	if err != nil {
		return nil
	}

	return fd
}

func (s *Server) tryVariantFile(w http.ResponseWriter, r *http.Request, ext, contentType string) (string, bool) {
	s.Debugf("trying variant file %s for %s", ext, r.URL.Path)
	filename := "index" + ext
	fixedPath := path.Clean(r.URL.Path)

	if !strings.HasSuffix(fixedPath, filename) {
		fixedPath = path.Join(fixedPath, filename)
		s.Debugf("added variant file to url: %s", fixedPath)
	}

	s.Debugf("checking if variant file exists: %s", fixedPath)
	_, err := s.fs.Stat(fixedPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.Debugf("variant file does not exist: %s", fixedPath)
		} else {
			s.Debugf("variant file error while stating: %s: %s", fixedPath, err)
		}
		return fixedPath, false
	}

	r.URL.Path = fixedPath
	s.Debugf("variant file exists: %s", r.URL.Path)
	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	return fixedPath, true
}

func (s *Server) tryActivity(w http.ResponseWriter, r *http.Request) {
	s.tryVariantFile(w, r, activityExt, activityContentType)
}

func (s *Server) tryMf2(w http.ResponseWriter, r *http.Request) {
	fixedPath, found := s.tryVariantFile(w, r, mf2Ext, mf2ContentType)
	if found {
		return
	}

	buildFor := path.Dir(fixedPath)
	s.Debugf("build mf2 for: %s", buildFor)

	fd := s.getHTML(buildFor)
	if fd == nil {
		return
	}
	defer fd.Close()

	abs, err := url.Parse(path.Dir(fixedPath))
	if err != nil {
		s.Warnf("could not parse url: %s", path.Dir(fixedPath))
		return
	}

	data := microformats.Parse(fd, abs)
	if data == nil {
		s.Warnf("microformats returned empty for: %s", fixedPath)
		return
	}

	bytes, err := json.Marshal(data)
	if data == nil {
		s.Warnf("could not marshal microformats: %s", err)
		return
	}

	err = afero.WriteFile(s.fs, fixedPath, bytes, 0644)
	if err != nil {
		s.Warnf("could not write file: %s", err)
		return
	}

	r.URL.Path = fixedPath
	w.Header().Set("Content-Type", mf2ContentType+"; charset=utf-8")
}

func (s *Server) staticHandler() http.HandlerFunc {
	domain, err := url.Parse(s.c.Domain)
	if err != nil {
		panic("domain is invalid")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		acceptsHTML := strings.Contains(accept, "text/html")
		acceptsActivity := strings.Contains(accept, activityContentType)
		acceptsMf2 := strings.Contains(accept, mf2ContentType)

		r.URL.Scheme = domain.Scheme
		r.URL.Host = domain.Host

		if strings.HasSuffix(r.URL.Path, "index.as2") || (!acceptsHTML && acceptsActivity) {
			s.tryActivity(w, r)
		}

		if strings.HasSuffix(r.URL.Path, "index.mf2") || (!acceptsHTML && acceptsMf2) {
			s.tryMf2(w, r)
		}

		nfw := &notFoundRedirectRespWr{ResponseWriter: w}
		s.httpdir.ServeHTTP(nfw, r)

		if nfw.status == http.StatusNotFound {
			w.Header().Del("Content-Type") // Let http.ServeFile set the correct header
			http.ServeFile(w, r, path.Join(s.c.Hugo.Destination, "404.html"))
		}
	}
}
