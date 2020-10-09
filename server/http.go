package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
)

// NOTE: instead of having many functions returning http.handleFunc, maybe I can
// have a global Server struct that has all handlers associated. That way, I don't
// need to pass the data to everything.

func Start(c *config.Config, s *services.Services) error {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	// auth := indieauth.With(&c.IndieAuth)

	r.Get("/micropub", getMicropubHandler(s, c))
	// r.With(auth).Get("/micropub", getMicropubHandler(s, c))
	r.Post("/micropub", postMicropubHandler(s, c))
	// r.With(auth).Post("/micropub", postMicropubHandler(s, c))

	r.Post("/webhook", webhookHandler(s, c))
	r.Post("/webmention", webmentionHandler(s, c))
	r.Post("/activitypub/inbox", activityPubInboxHandler(s, c))

	r.NotFound(staticHandler(c))
	r.MethodNotAllowed(staticHandler(c))

	// NOTE:
	//	- Should I handle /now dynamicall?
	//	- Should I handle all redirects dynamically?

	return http.ListenAndServe(":"+strconv.Itoa(c.Port), r)
}

func serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("error while serving json: %s", err)
	}
}

func serveError(w http.ResponseWriter, code int, err error) {
	serveJSON(w, code, map[string]interface{}{
		"error":             http.StatusText(code),
		"error_description": err.Error(),
	})
}
