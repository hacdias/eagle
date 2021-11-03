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
			s.serveErrorHTML(w, r, http.StatusNotFound, nil)
			return
		}
		path = filepath.Join(s.Config.SourceDirectory, path)
		http.ServeFile(w, r, path)
		return
	}

	s.entryGet(w, r)
}

func (s *Server) wildcardPost(w http.ResponseWriter, r *http.Request) {

	w.Write([]byte("unsupported for now"))
}
