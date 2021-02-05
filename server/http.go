package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

func (s *Server) StartHTTP() error {
	r := chi.NewRouter()
	r.Use(s.recoverer)

	r.Post("/webhook", s.webhookHandler)
	r.Post("/webmention", s.webmentionHandler)
	r.Post("/activitypub/inbox", s.activityPubPostInboxHandler)
	r.Get("/search.json", s.searchHandler)

	// Make sure we have a built version!
	should, err := s.Hugo.ShouldBuild()
	if err != nil {
		return err
	}
	if should {
		err = s.Hugo.Build(false)
		if err != nil {
			return err
		}
	}

	static := s.staticHandler()

	r.NotFound(static)
	r.MethodNotAllowed(static)

	// NOTE:
	//	- Should I handle /now dynamicall?
	//	- Should I handle all redirects dynamically?

	s.Infof("Listening on http://localhost:%d", s.c.Port)
	return http.ListenAndServe(":"+strconv.Itoa(s.c.Port), r)
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
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
