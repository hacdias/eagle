package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.hacdias.com/eagle/log"
)

func (s *Server) makeRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(s.withRecoverer)
	r.Use(log.WithZap)

	// middleware.CleanPath

	r.Use(withCleanPath)
	r.Use(middleware.GetHead)
	r.Use(withSecurityHeaders)

	r.Get("/*", s.everythingBagel)
	return r
}

func (s *Server) everythingBagel(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(r.URL.Path))
}
