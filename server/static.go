package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/fs"
)

func (s *Server) withRedirects(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if url, ok := s.redirects[r.URL.Path]; ok {
			http.Redirect(w, r, url, http.StatusMovedPermanently)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func setCacheAsset(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
}

func setCacheHTML(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0")
}

func setCacheDefault(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=15552000")
}

func (s *Server) serveAssets(w http.ResponseWriter, r *http.Request) {
	if asset := s.renderer.GetAssets().Get(r.URL.Path); asset != nil {
		setCacheAsset(w)
		w.Header().Set("Content-Type", asset.Type)
		_, _ = w.Write(asset.Body)
	} else {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	}
}

// TODO: right now, we are doing 2 FS checks for each entry.
// To improve this, we avoid handling paths that do not have extensions. However,
// I don't really like the way this is done and I wonder if this could be improved.
func (s *Server) withStaticFiles(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := filepath.Ext(r.URL.Path)
		if ext == "" {
			next.ServeHTTP(w, r)
			return
		}

		filename := filepath.Join(s.c.Source.Directory, fs.ContentDirectory, r.URL.Path)
		if stat, err := os.Stat(filename); err == nil && stat.Mode().IsRegular() {
			// Do not serve .* (dot)files.
			if strings.HasPrefix(stat.Name(), ".") {
				s.serveErrorHTML(w, r, http.StatusNotFound, nil)
				return
			}

			f, err := os.Open(filename)
			if err != nil {
				s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
				return
			}
			defer f.Close()

			setCacheDefault(w)
			http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
			return
		}

		next.ServeHTTP(w, r)
	})
}
