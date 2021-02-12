package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

const activityContentType = "application/activity+json"
const activityExt = ".as2"

func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: see if removing this improves speed,
	// This works and improved the speed substaintialy. Find other was to improve speed.
	// s.staticFsLock.RLock()
	// defer s.staticFsLock.RUnlock()

	accept := r.Header.Get("Accept")
	acceptsHTML := strings.Contains(accept, "text/html")
	acceptsActivity := strings.Contains(accept, activityContentType)

	if strings.HasSuffix(r.URL.Path, "index.as2") || (!acceptsHTML && acceptsActivity) {
		s.tryActivity(w, r)
	}

	nfw := &notFoundRedirectRespWr{ResponseWriter: w}
	s.staticFs.ServeHTTP(nfw, r)

	if nfw.status == http.StatusNotFound {
		w.Header().Del("Content-Type") // Let http.ServeFile set the correct header
		r.URL.Path = "/404.html"
		s.staticFs.ServeHTTP(w, r)
	}
}

func (s *Server) tryActivity(w http.ResponseWriter, r *http.Request) {
	fixedPath := path.Clean(r.URL.Path)
	if !strings.HasSuffix(fixedPath, activityExt) {
		fixedPath = path.Join(fixedPath, "index"+activityExt)
	}

	// Locked by caller.
	_, err := s.staticFs.Stat(fixedPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.Infow("activity file does not exist", "path", fixedPath)
		} else {
			s.Warnf("error while stat'ing", "path", fixedPath, "error", err)
		}
		return
	}

	r.URL.Path = fixedPath
	w.Header().Set("Content-Type", activityContentType+"; charset=utf-8")
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

func (s *staticFs) readAS2(filepath string) (map[string]interface{}, error) {
	if !strings.HasSuffix(filepath, ".as2") {
		filepath = path.Join(filepath, "index.as2")
	}

	fd, err := s.Fs.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	var m map[string]interface{}
	err = json.NewDecoder(fd).Decode(&m)
	return m, err
}
