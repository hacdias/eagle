package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/indieauth"
	"github.com/hacdias/eagle/services"
)

type Server struct {
	*services.Services
	Config *config.Config
}

func Start(c *config.Config, s *services.Services) error {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	auth := indieauth.With(&c.IndieAuth)

	r.With(auth).Get("/micropub", getMicropubHandler(s, c))
	r.With(auth).Post("/micropub", postMicropubHandler(s, c))

	r.Post("/webhook", webhookHandler(s, c))
	r.Post("/webmention", webmentionHanndler(s, c))
	r.Post("/activitypub/inbox", activityPubInboxHandler(s, c))

	r.NotFound(staticHandler(c.Hugo.Destination))
	r.MethodNotAllowed(staticHandler(c.Hugo.Destination))

	return http.ListenAndServe(":"+strconv.Itoa(c.Port), r)
}

func serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func serveError(w http.ResponseWriter, code int, err error) {
	serveJSON(w, code, map[string]interface{}{
		"error":             http.StatusText(code),
		"error_description": err.Error(),
	})
}
