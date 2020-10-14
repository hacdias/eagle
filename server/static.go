package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const activityContentType = "application/activity+json"
const activityExt = ".as2"

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

func (s *Server) getAS2(url string) (map[string]interface{}, error) {
	if !strings.HasSuffix(url, ".as2") {
		url = path.Join(url, "index.as2")
	}

	fd, err := s.fs.Open(url)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	var m map[string]interface{}
	err = json.NewDecoder(fd).Decode(&m)
	return m, err
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

func (s *Server) staticHandler() http.HandlerFunc {
	domain, err := url.Parse(s.c.Domain)
	if err != nil {
		panic("domain is invalid")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		acceptsHTML := strings.Contains(accept, "text/html")
		acceptsActivity := strings.Contains(accept, activityContentType)

		r.URL.Scheme = domain.Scheme
		r.URL.Host = domain.Host

		if strings.HasSuffix(r.URL.Path, "index.as2") || (!acceptsHTML && acceptsActivity) {
			s.tryActivity(w, r)
		}

		nfw := &notFoundRedirectRespWr{ResponseWriter: w}
		s.httpdir.ServeHTTP(nfw, r)

		if nfw.status == http.StatusNotFound {
			w.Header().Del("Content-Type") // Let http.ServeFile set the correct header
			r.URL.Path = "/404.html"
			s.httpdir.ServeHTTP(w, r)
		}
	}
}

type neuteredFs struct {
	http.FileSystem
}

func (nfs neuteredFs) Open(path string) (http.File, error) {
	f, err := nfs.FileSystem.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := nfs.FileSystem.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}

			return nil, err
		}
	}

	return f, nil
}
