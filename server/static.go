package server

import (
	"net/http"
	"path/filepath"
)

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

func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	nfw := &notFoundRedirectRespWr{ResponseWriter: w}
	s.staticHTTP.ServeHTTP(nfw, r)

	if nfw.status == http.StatusNotFound {
		w.Header().Del("Content-Type")
		s.notFoundHandler(w, r)
	}
}
