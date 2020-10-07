package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hacdias/eagle/config"
)

func Start(cfg *config.Config) error {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/micropub", getMicropubHandler)
	r.Post("/micropub", postMicropubHandler)
	r.Post("/webhook", todo)
	r.Post("/webmention", todo)

	r.NotFound(staticHandler(cfg.Hugo.Destination))
	r.MethodNotAllowed(staticHandler(cfg.Hugo.Destination))

	return http.ListenAndServe(":"+strconv.Itoa(cfg.Port), r)
}

func serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func todo(w http.ResponseWriter, r *http.Request) {
	serveJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"message": "this endpoint is planned, but not yet implemented",
	})
}
