package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth"
)

func (s *Server) makeRouter(noDashboard bool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.CleanPath)
	r.Use(middleware.RedirectSlashes)
	r.Use(middleware.GetHead)

	r.Use(s.recoverer)
	r.Use(s.securityHeaders)

	if !noDashboard {
		if s.Config.Auth != nil {
			r.Use(jwtauth.Verifier(s.token))
		}

		r.Use(s.isAuthenticated)
	}

	// if s.c.Development {
	r.Get("/micropub", s.getMicropubHandler)
	r.Post("/micropub", s.postMicropubHandler)
	// } else {
	// 	auth := indieauth.With(&s.c.IndieAuth, s.Named("indieauth"))
	// 	r.With(auth).Get("/micropub", s.getMicropubHandler)
	// 	r.With(auth).Post("/micropub", s.postMicropubHandler)
	// }

	if s.Config.Tor != nil {
		r.Use(s.onionHeader)
		r.Get("/onion", s.onionRedirHandler)
	}

	r.Get("/search.json", s.searchHandler)

	// if s.Config.WebhookSecret != "" {
	// 	r.Post("/webhook", s.webhookHandler)
	// }

	if s.Config.WebmentionsSecret != "" {
		r.Post("/webmention", s.webmentionHandler)
	}

	r.With(s.withEntry).Get("/*", s.entryHandler)
	// r.NotFound(s.staticHandler)         // NOTE: maybe repetitive regarding previous line.
	// r.MethodNotAllowed(s.staticHandler) // NOTE: maybe useless.

	// if noDashboard {
	// 	return r
	// }

	// r.Route(dashboardPath, func(r chi.Router) {
	// 	fs := http.FS(static.FS)
	// 	if s.Config.Development {
	// 		fs = http.FS(afero.NewIOFS(afero.NewBasePathFs(afero.NewOsFs(), "./dashboard/static")))
	// 	}

	// 	httpdir := http.FileServer(neuteredFs{fs})

	// 	r.Group(func(r chi.Router) {
	// 		if s.Config.Auth != nil {
	// 			r.Use(s.mustAuthenticate)
	// 		}

	// 		r.Get("/", s.dashboardGetHandler)
	// 		r.Get("/new", s.newGetHandler)
	// 		r.Get("/reply", s.replyGetHandler)
	// 		r.Get("/edit*", s.editGetHandler)
	// 		r.Get("/webmentions*", s.webmentionsGetHandler)
	// 		r.Get("/blogroll", s.blogrollGetHandler)
	// 		r.Get("/gedit", s.geditGetHandler)
	// 		r.Get("/sync", s.syncGetHandler)
	// 		r.Get("/rebuild-index", s.rebuildIndexGetHandler)

	// 		r.Post("/new", s.newPostHandler)
	// 		r.Post("/edit*", s.editPostHandler)
	// 		r.Post("/webmentions", s.webmentionsPostHandler)
	// 		r.Post("/gedit", s.geditPostHandler)
	// 	})

	// 	r.Get("/*", http.StripPrefix(dashboardPath, httpdir).ServeHTTP)
	// })

	// if s.Config.Auth != nil {
	// 	r.Get("/logout", s.logoutGetHandler)
	// 	r.Get("/login", s.loginGetHandler)
	// 	r.Post("/login", s.loginPostHandler)
	// }

	return r
}
