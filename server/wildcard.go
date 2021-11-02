package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v2/eagle"
)

func (s *Server) wildcardGet(w http.ResponseWriter, r *http.Request) {
	// TODO: find better solution for this
	staticFile := filepath.Join(s.Config.SourceDirectory, eagle.StaticDirectory, r.URL.Path)
	if stat, err := os.Stat(staticFile); err == nil && stat.Mode().IsRegular() {
		http.ServeFile(w, r, staticFile)
		return
	}

	// TODO: find better solution for this. Asset fioles may need to be built.
	assetFile := filepath.Join(s.Config.SourceDirectory, eagle.AssetsDirectory, r.URL.Path)
	if stat, err := os.Stat(assetFile); err == nil && stat.Mode().IsRegular() {
		http.ServeFile(w, r, assetFile)
		return
	}

	path := filepath.Join(eagle.ContentDirectory, r.URL.Path)
	path = filepath.Clean(path)

	if stat, err := s.SrcFs.Stat(path); err == nil && stat.Mode().IsRegular() {
		if strings.Contains(path, "/private/") {
			s.serveError(w, http.StatusNotFound, nil)
			return
		}
		path = filepath.Join(s.Config.SourceDirectory, path)
		http.ServeFile(w, r, path)
		return
	}

	entry, err := s.GetEntry(r.URL.Path)
	if os.IsNotExist(err) {
		s.serveError(w, http.StatusNotFound, nil)
		return
	} else if err != nil {
		s.serveError(w, http.StatusInternalServerError, err)
		return
	}

	tpls := []string{}
	if entry.Section != "" {
		tpls = append(tpls, eagle.TemplateSingle+"."+entry.Section)
	}
	tpls = append(tpls, eagle.TemplateSingle)

	s.render(w, &eagle.RenderData{
		Entry: entry,
	}, tpls)
}

func (s *Server) wildcardPost(w http.ResponseWriter, r *http.Request) {

	w.Write([]byte("unsupported for now"))
}
