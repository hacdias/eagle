package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
)

func (s *Server) makeRouter() http.Handler {
	r := chi.NewRouter()

	if s.c.Server.Logging || s.c.Development {
		r.Use(middleware.Logger)
	}

	r.Use(withCleanPath)
	r.Use(middleware.GetHead)
	r.Use(s.withRecoverer)
	r.Use(s.withSecurityHeaders)
	r.Use(jwtauth.Verifier(s.jwtAuth))
	r.Use(s.withLoggedIn)

	// GitHub WebHook
	if s.c.Server.WebhookSecret != "" {
		r.Post("/webhook", s.webhookPost)
	}

	// Webmentions Handler
	if s.c.Webmentions.Secret != "" {
		r.Post("/webmention", s.webmentionPost)
	}

	// Random
	r.Get("/search/", s.searchGet)
	r.Get("/.well-known/webfinger", s.webFingerGet)
	r.Post("/guestbook/", s.guestbookPost)

	// Login
	r.Get("/login", s.loginGet)
	r.Post("/login", s.loginPost)
	r.Get("/logout", s.logoutGet)

	// IndieAuth Server (Part I)
	r.Get("/.well-known/oauth-authorization-server", s.indieauthGet)
	r.Post("/auth", s.authPost)
	r.Post("/token", s.tokenPost)
	r.Post("/token/verify", s.tokenVerifyPost)

	// Admin only pages.
	r.Group(func(r chi.Router) {
		r.Use(s.mustLoggedIn)

		// IndieAuth Server (Part II)
		r.Get("/auth", s.authGet)
		r.Post("/auth/accept", s.authAcceptPost)

		r.Get("/new", s.newGet)
		r.Post("/new", s.newPost)

		r.Get("/edit*", s.editGet)
		r.Post("/edit*", s.editPost)

		r.Get("/dashboard", s.dashboardGet)
		r.Post("/dashboard", s.dashboardPost)

		r.Get("/deleted", s.deletedGet)
		r.Get("/drafts", s.draftsGet)
		r.Get("/unlisted", s.unlistedGet)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.mustIndieAuth)

		// IndieAuth Server
		r.Get("/token", s.tokenGet) // Backwards compatible token verification endpoint
		r.Get("/userinfo", s.userInfoGet)
	})

	// Everything that was not matched so far.
	r.Group(func(r chi.Router) {
		r.Use(s.withRedirects)

		r.Get("/*", s.staticHandler)
		r.NotFound(s.staticHandler)         // NOTE: maybe repetitive regarding previous line.
		r.MethodNotAllowed(s.staticHandler) // NOTE: maybe useless.
	})

	return r
}
