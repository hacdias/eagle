package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth"
)

func (s *Server) makeRouter(noDashboard bool) http.Handler {
	r := chi.NewRouter()

	// r.Use(middleware.Logger)
	r.Use(middleware.RedirectSlashes)
	r.Use(cleanPath)
	r.Use(middleware.GetHead)

	r.Use(s.recoverer)
	r.Use(s.securityHeaders)

	if !noDashboard {
		if s.Config.Auth != nil {
			r.Use(jwtauth.Verifier(s.token))
		}

		r.Use(s.isAuthenticated)
	}

	r.Group(func(r chi.Router) {
		// TODO: Protect with IndieAuth

		r.Get("/micropub", s.micropubGet)
		r.Post("/micropub", s.micropubPost)
	})

	r.Group(func(r chi.Router) {
		// TODO: Protect with Login

		r.Get("/new", s.newGet)
		r.Post("/new", s.newPost)
	})

	r.Group(s.listingRoutes)

	if s.Config.Tor != nil {
		r.Use(s.onionHeader)
		r.Get("/onion", s.onionRedirHandler)
	}

	// if s.Config.WebhookSecret != "" {
	// 	r.Post("/webhook", s.webhookHandler)
	// }

	if s.Config.WebmentionsSecret != "" {
		r.Post("/webmention", s.webmentionHandler)
	}

	// r.Get("/tags")

	r.Get("/*", s.wildcardGet)
	r.Post("/*", s.wildcardPost)

	return r
}

const (
	feedPath  = ".{feed:xml|json}"
	yearPath  = `/{year:x|\d\d\d\d}` // TODO: fix
	monthPath = yearPath + `/{month:x|\d\d}`
	dayPath   = monthPath + `/{day:x|\d\d}`
)

func (s *Server) listingRoutes(r chi.Router) {
	r.Get("/", s.indexGet)
	r.Get("/feed"+feedPath, s.indexGet)

	r.Get(yearPath, s.dateGet)
	r.Get(yearPath+feedPath, s.dateGet)
	r.Get(monthPath, s.dateGet)
	r.Get(monthPath+feedPath, s.dateGet)
	r.Get(dayPath, s.dateGet)
	r.Get(dayPath+feedPath, s.dateGet)

	r.Get("/tags/{tag}", s.tagGet)
	r.Get("/tags/{tag}"+feedPath, s.tagGet)
	r.Get("/search", s.searchGet)

	for _, section := range s.Config.Site.Sections {
		r.Get("/"+section, s.sectionGet(section))
		r.Get("/"+section+feedPath, s.sectionGet(section))
	}
}

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
