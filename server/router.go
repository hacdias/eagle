package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/hacdias/eagle/dashboard/static"
	"github.com/spf13/afero"
)

func (s *Server) makeRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(s.recoverer)
	r.Use(s.headers)

	if s.c.Auth != nil {
		r.Use(jwtauth.Verifier(s.token))
	}

	r.Use(s.isAuthenticated)

	r.Get("/search.json", s.searchHandler)
	r.Post("/webhook", s.webhookHandler)
	r.Post("/webmention", s.webmentionHandler)

	r.Get("/*", s.staticHandler)
	r.NotFound(s.staticHandler)         // NOTE: maybe repetitive regarding previous line.
	r.MethodNotAllowed(s.staticHandler) // NOTE: maybe useless.

	r.Route(dashboardPath, func(r chi.Router) {
		fs := http.FS(static.FS)
		if s.c.Development {
			fs = http.FS(afero.NewIOFS(afero.NewBasePathFs(afero.NewOsFs(), "./dashboard/static")))
		}

		httpdir := http.FileServer(neuteredFs{fs})

		r.Group(func(r chi.Router) {
			if s.c.Auth != nil {
				r.Use(s.mustAuthenticate)
			}

			r.Get("/", s.dashboardGetHandler)
			r.Get("/new", s.newGetHandler)
			r.Get("/edit", s.editGetHandler)
			r.Get("/reply", s.replyGetHandler)
			r.Get("/delete", s.deleteGetHandler)
			r.Get("/reshare", s.reshareGetHandler)
			r.Get("/blogroll", s.blogrollGetHandler)
			r.Get("/gedit", s.geditGetHandler)

			r.Post("/", s.dashboardPostHandler)
			r.Post("/new", s.newPostHandler)
			r.Post("/edit", s.editPostHandler)
			r.Post("/delete", s.deletePostHandler)
			r.Post("/reshare", s.resharePostHandler)
			r.Post("/gedit", s.geditPostHandler)
		})

		r.Get("/*", http.StripPrefix(dashboardPath, httpdir).ServeHTTP)
	})

	if s.c.Auth != nil {
		r.Get("/logout", s.logoutGetHandler)
		r.Get("/login", s.loginGetHandler)
		r.Post("/login", s.loginPostHandler)
	}

	return r
}