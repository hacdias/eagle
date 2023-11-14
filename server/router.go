package server

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.hacdias.com/eagle/log"
	"go.hacdias.com/eagle/render"
)

func (s *Server) makeRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(s.withRecoverer)
	r.Use(log.WithZap)

	// middleware.CleanPath

	r.Use(withCleanPath)
	r.Use(middleware.GetHead)
	r.Use(withSecurityHeaders)

	r.Use(s.withStaticFiles)

	r.Get(render.AssetsBaseURL+"*", s.serveAssets)

	r.Get("/*", s.everythingBagel)
	return r
}

func (s *Server) everythingBagel(w http.ResponseWriter, r *http.Request) {
	e, err := s.fs.GetEntry(r.URL.Path)
	if os.IsNotExist(err) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	// TODO: if e.Deleted() {
	// 	s.serveErrorHTML(w, r, http.StatusGone, nil)
	// 	return
	// }

	if e.Draft {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	if r.URL.Path != e.ID {
		http.Redirect(w, r, e.ID, http.StatusTemporaryRedirect)
		return
	}

	_ = s.renderer.Render(w, e)
}

func (s *Server) withStaticFiles(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := path.Ext(r.URL.Path)
		if ext == "" {
			next.ServeHTTP(w, r)
			return
		}

		if stat, err := s.fs.ContentFS.Stat(r.URL.Path); err == nil && stat.Mode().IsRegular() {
			// Do not serve (dot)files.
			if strings.HasPrefix(stat.Name(), ".") {
				s.serveErrorHTML(w, r, http.StatusNotFound, nil)
				return
			}

			f, err := s.fs.ContentFS.Open(r.URL.Path)
			if err != nil {
				s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
				return
			}
			defer f.Close()

			// TODO: setCacheDefault(w)
			http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) serveAssets(w http.ResponseWriter, r *http.Request) {
	if asset := s.renderer.AssetByPath(r.URL.Path); asset != nil {
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Header().Set("Content-Type", asset.Type)
		_, _ = w.Write(asset.Body)
	} else {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	}
}
