package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v2/eagle"
)

func (s *Server) withStaticFiles(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(v2): different router?
		if url, ok := s.redirects[r.URL.Path]; ok {
			http.Redirect(w, r, url, http.StatusMovedPermanently)
			return
		}

		// TODO(v2): find better solution for this
		staticFile := filepath.Join(s.Config.SourceDirectory, eagle.StaticDirectory, r.URL.Path)
		if stat, err := os.Stat(staticFile); err == nil && stat.Mode().IsRegular() {
			http.ServeFile(w, r, staticFile)
			return
		}

		// TODO(v2): build assets
		// TODO(v2): find better solution for this. Asset fioles may need to be built.
		assetFile := filepath.Join(s.Config.SourceDirectory, eagle.AssetsDirectory, r.URL.Path)
		if stat, err := os.Stat(assetFile); err == nil && stat.Mode().IsRegular() {
			http.ServeFile(w, r, assetFile)
			return
		}

		// TODO(v2): do not do this
		contentFile := filepath.Join(s.Config.SourceDirectory, eagle.ContentDirectory, r.URL.Path)
		if stat, err := os.Stat(contentFile); err == nil && stat.Mode().IsRegular() {
			if strings.HasPrefix(stat.Name(), "_") {
				// Do not serve _* files.
				s.serveErrorHTML(w, r, http.StatusNotFound, nil)
				return
			}
			http.ServeFile(w, r, contentFile)
			return
		}

		next.ServeHTTP(w, r)
	})
}
