package server

import (
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/afero"
)

func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: see if removing this improves speed,
	// This works and improved the speed substaintialy. Find other was to improve speed.
	// s.staticFsLock.RLock()
	// defer s.staticFsLock.RUnlock()
	nfw := &notFoundRedirectRespWr{ResponseWriter: w}
	s.staticFs.ServeHTTP(nfw, r)

	if nfw.status == http.StatusNotFound {
		bytes, err := afero.ReadFile(s.staticFs.Fs, "404.html")
		if err != nil {
			s.serveError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "text/html; charset=utf-8") // Let http.ServeFile set the correct header
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(bytes)
	}
}

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

type staticFs struct {
	dir string
	afero.Fs
	http.Handler
}

func newStaticFs(dir string) *staticFs {
	fs := afero.NewBasePathFs(afero.NewOsFs(), dir)
	handler := http.FileServer(neuteredFs{afero.NewHttpFs(fs).Dir("/")})

	return &staticFs{
		dir:     dir,
		Fs:      fs,
		Handler: handler,
	}
}

func (s *staticFs) readHTML(filepath string) ([]byte, error) {
	if !strings.HasSuffix(filepath, ".html") {
		filepath = path.Join(filepath, "index.html")
	}

	return afero.ReadFile(s, filepath)
}
