package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
)

func Start(c *config.Config) error {
	s := services.NewServices(c)
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	// auth := indieauth.With(&c.IndieAuth)

	// r.With(auth).Get("/micropub", getMicropubHandler(s, c))
	r.Get("/micropub", getMicropubHandler(s, c))
	// r.With(auth).Post("/micropub", postMicropubHandler(s, c))
	r.Post("/micropub", postMicropubHandler(s, c))
	r.Post("/webhook", todo)
	r.Post("/webmention", todo)

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

func todo(w http.ResponseWriter, r *http.Request) {
	serveJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"message": "this endpoint is planned, but not yet implemented",
	})
}
