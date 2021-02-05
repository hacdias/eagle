package server

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/hacdias/eagle/eagle"
)

func (s *Server) StartHTTP() error {
	r := chi.NewRouter()
	r.Use(s.recoverer)

	// r.Post("/webhook", s.webhookHandler)
	// r.Post("/webmention", s.webmentionHandler)
	// r.Post("/activitypub/inbox", s.activityPubPostInboxHandler)
	// r.Get("/search.json", s.searchHandler)

	// Make sure we have a built version!
	//should, err := s.Hugo.ShouldBuild()
	//if err != nil {
	//	return err
	//}
	//if should {
	//	err = s.Hugo.Build(false)
	//	if err != nil {
	//		return err
	//	}
	//}

	//s//tatic := s.staticHandler()
	//
	//r.NotFound(static)
	//.MethodNotAllowed(static)

	// NOTE:
	//	- Should I handle /now dynamicall?
	//	- Should I handle all redirects dynamically?

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		urlPath := path.Clean(r.URL.Path)
		if r.URL.Path != urlPath {
			http.Redirect(w, r, urlPath, http.StatusTemporaryRedirect)
			return
		}

		filePath := filepath.Join(s.c.Source, "content", urlPath+".md")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		e, _ := eagle.NewEagle(s.c)
		entry, _ := e.GetEntry(urlPath)

		fmt.Println(entry.Metadata)

		err := s.RenderHTML(entry, w)
		if err != nil {
			s.Error(err)
		}
	})

	s.Infof("Listening on http://localhost:%d", s.c.Port)
	return http.ListenAndServe(":"+strconv.Itoa(s.c.Port), r)
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Middlware is working??")
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				s.Errorf("panic while serving: %s", rvr)
				s.Notify.Error(fmt.Errorf(fmt.Sprint(rvr)))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
