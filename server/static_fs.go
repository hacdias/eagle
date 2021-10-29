package server

import (
	"net/http"
	"path/filepath"

	"github.com/spf13/afero"
)

type staticFs struct {
	http.Handler
	afero.Afero
	dir string
}

func newStaticFs(dir string) *staticFs {
	fs := afero.NewBasePathFs(afero.NewOsFs(), dir)
	httpFs := neuteredFs{afero.NewHttpFs(fs).Dir("/")}
	handler := http.FileServer(httpFs)

	return &staticFs{
		Handler: handler,
		Afero:   afero.Afero{Fs: fs},
		dir:     dir,
	}
}

// neuteredFs is a file system that returns 404 when a directory contains
//  no index.html to prevent http.FileServer to render a listing of the directory.
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
