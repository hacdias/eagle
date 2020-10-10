package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/middleware/indieauth"
	"github.com/hacdias/eagle/services"
	"go.uber.org/zap"
)

// NOTE: instead of having many functions returning http.handleFunc, maybe I can
// have a global Server struct that has all handlers associated. That way, I don't
// need to pass the data to everything.

func Start(log *zap.SugaredLogger, c *config.Config, s *services.Services) error {
	server := Server{
		SugaredLogger: log,
		Services:      s,
		c:             c,
	}

	r := chi.NewRouter()

	if c.Development {
		r.Get("/micropub", server.getMicropubHandler)
		r.Post("/micropub", server.postMicropubHandler)
	} else {
		auth := indieauth.With(&c.IndieAuth, log.Named("indieauth"))
		r.With(auth).Get("/micropub", server.getMicropubHandler)
		r.With(auth).Post("/micropub", server.postMicropubHandler)
	}

	r.Post("/webhook", server.webhookHandler)
	r.Post("/webmention", server.webmentionHandler)
	r.Post("/activitypub/inbox", server.activityPubInboxHandler)

	static := server.staticHandler()

	r.NotFound(static)
	r.MethodNotAllowed(static)

	// NOTE:
	//	- Should I handle /now dynamicall?
	//	- Should I handle all redirects dynamically?

	log.Infof("Listening on http://localhost:%d", c.Port)
	return http.ListenAndServe(":"+strconv.Itoa(c.Port), r)
}
