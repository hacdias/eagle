package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/hacdias/eagle/middleware/indieauth"
)

func (s *Server) StartHTTP() error {
	r := chi.NewRouter()
	r.Use(s.recoverer)

	if s.c.Development {
		r.Get("/micropub", s.getMicropubHandler)
		r.Post("/micropub", s.postMicropubHandler)
	} else {
		auth := indieauth.With(&s.c.IndieAuth, s.Named("indieauth"))
		r.With(auth).Get("/micropub", s.getMicropubHandler)
		r.With(auth).Post("/micropub", s.postMicropubHandler)
	}

	r.Post("/webhook", s.webhookHandler)
	r.Post("/webmention", s.webmentionHandler)
	r.Post("/activitypub/inbox", s.activityPubInboxHandler)
	r.Get("/search.json", s.searchHandler)

	// Make sure we have a built version!
	err := s.Hugo.Build(false)
	if err != nil {
		return err
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
