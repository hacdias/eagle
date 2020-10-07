package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hacdias/eagle/config"
)

func dummy(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("dummy"))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.Write([]byte("404"))
}

func pong(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func serveJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func Start(cfg *config.Config) {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Route("/eagle", func(r chi.Router) {
		r.Get("/micropub", getMicropubHandler)
		r.Post("/micropub", postMicropubHandler)
		r.Get("/ping", pong)
		r.Post("/webhook", dummy)
		r.Post("/webmention", dummy)
	})

	r.NotFound(notFound)
	r.MethodNotAllowed(notFound)

	http.ListenAndServe(":3000", r)
}
