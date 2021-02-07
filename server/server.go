package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
	"github.com/spf13/afero"
	"go.uber.org/zap"
)

type Server struct {
	*zap.SugaredLogger
	*services.Services
	c          *config.Config
	router     *mux.Router
	staticHTTP http.Handler
	layouts    map[string]*template.Template
}

func NewServer(c *config.Config, s *services.Services) (*Server, error) {
	layouts, err := getTemplates(c.Source)
	if err != nil {
		return nil, err
	}

	staticDir := filepath.Join(c.Source, "static")
	staticFs := afero.NewBasePathFs(afero.NewOsFs(), staticDir)
	staticHTTP := http.FileServer(neuteredFs{afero.NewHttpFs(staticFs).Dir("/")})

	server := &Server{
		SugaredLogger: c.S().Named("server"),
		Services:      s,
		c:             c,
		staticHTTP:    staticHTTP,
		layouts:       layouts,
	}

	fmt.Println(layouts)

	r := mux.NewRouter()

	r.Use(server.recoverer)
	// r.Use(server.cleanPath)

	// TODO: do not forget to setup redirects for feeds and update the links
	// in the /follow page.

	// This are the pages that have pagination.
	// TODO: consider removing the /all page altogether and moving the feed
	// to the root of the domain.
	for _, listing := range []string{"micro", "all"} {
		r.HandleFunc(fmt.Sprintf("/%s", listing), server.todoHandler)
		r.HandleFunc(fmt.Sprintf("/%s/page/{page:[0-9]+}", listing), server.todoHandler)
		r.HandleFunc(fmt.Sprintf("/%s/page/{page:[0-9]+}.rss", listing), server.todoHandler)
		r.HandleFunc(fmt.Sprintf("/%s/page/{page:[0-9]+}.json", listing), server.todoHandler)
	}

	for _, listing := range []string{"articles", "watches", "darkroom", "notes"} {
		r.HandleFunc(fmt.Sprintf("/%s", listing), server.todoHandler)
	}

	// Tags pages!
	r.HandleFunc("/tags/{tag}", server.todoHandler)
	r.HandleFunc("/tags/{tag}/page/{page:[0-9]+}", server.todoHandler)

	// All other pages (check on the fly!)
	r.PathPrefix("/").HandlerFunc(server.mainHandler).Methods("GET", "HEAD")

	// r.NotFoundHandler =
	// r.MethodNotAllowedHandler =

	server.router = r
	return server, nil
}

func (s *Server) StartHTTP() error {
	s.Infof("Listening on http://localhost:%d", s.c.Port)
	return http.ListenAndServe(":"+strconv.Itoa(s.c.Port), s.router)
}

func (s *Server) todoHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("TODO"))
}

func (s *Server) mainHandler(w http.ResponseWriter, r *http.Request) {
	ext := path.Ext(r.URL.Path)
	id := path.Clean(strings.TrimSuffix(r.URL.Path, ext))
	path := id

	accept := r.Header.Get("Accept")
	acceptsHTML := strings.Contains(accept, "text/html")
	acceptsActivity := strings.Contains(accept, activityContentType)
	serveActivity := ext == activityExt || (!acceptsHTML && acceptsActivity)

	if serveActivity {
		path += activityExt
	}

	entry, err := s.GetEntry(id)
	if os.IsNotExist(err) {
		s.staticHandler(w, r)
		return
	}

	if err != nil {
		s.serveHTMLError(w, http.StatusInternalServerError, err)
		return
	}

	if r.URL.Path != path {
		http.Redirect(w, r, path, http.StatusTemporaryRedirect)
		return
	}

	if serveActivity {
		// TODO: serve actual activity
		s.serveJSON(w, http.StatusOK, entry.Metadata)
	} else {
		s.serveHTML(w, entry)
	}
}

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	// TOOD: use template
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Page Not Found"))
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

func (s *Server) recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				s.Errorw("panic while serving", "path", r.URL.Path, "error", rvr)
				// TODO s.Notify.Error(fmt.Errorf(fmt.Sprint(rvr)))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
