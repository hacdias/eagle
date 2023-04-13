package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
)

func (s *Server) makeRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(s.withRecoverer)

	if s.c.Server.Logging || s.c.Development {
		r.Use(middleware.Logger)
	}

	r.Use(withCleanPath)
	r.Use(middleware.GetHead)
	r.Use(s.withSecurityHeaders)
	r.Use(jwtauth.Verifier(s.jwtAuth))
	r.Use(s.withLoggedIn)
	r.Use(s.withAdminBar)

	// GitHub WebHook
	if s.c.Server.WebhookSecret != "" {
		r.Post(webhookPath, s.webhookPost)
	}

	// Webmentions Handler
	if s.c.Webmentions.Secret != "" {
		r.Post(webmentionPath, s.webmentionPost)
	}

	// Random
	r.Get(webFingerPath, s.webFingerGet)
	r.Get("/search/", s.searchGet)
	r.Post("/guestbook/", s.guestbookPost)

	// Login
	r.Get(loginPath, s.loginGet)
	r.Post(loginPath, s.loginPost)
	r.Get(logoutPath, s.logoutGet)

	// IndieAuth Server (Part I)
	r.Get(wellKnownOAuthServer, s.indieauthGet)
	r.Post(authPath, s.authPost)
	r.Post(tokenPath, s.tokenPost)
	r.Post(tokenVerifyPath, s.tokenVerifyPost)

	// Admin only pages.
	r.Group(func(r chi.Router) {
		r.Use(s.mustLoggedIn)

		// IndieAuth Server (Part II)
		r.Get(authPath, s.authGet)
		r.Post(authAcceptPath, s.authAcceptPost)

		r.Get(dashboardPath, s.dashboardGet)
		r.Post(dashboardPath, s.dashboardPost)

		r.Get(newPath, s.newGet)
		r.Post(newPath, s.newPost)

		r.Get(editPath+"*", s.editGet)
		r.Post(editPath+"*", s.editPost)

		r.Get(deletedPath, s.deletedGet)
		r.Get(draftsPath, s.draftsGet)
		r.Get(unlistedPath, s.unlistedGet)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.mustIndieAuth)

		// IndieAuth Server
		r.Get(tokenPath, s.tokenGet) // Backwards compatible token verification endpoint
		r.Get(userInfoPath, s.userInfoGet)
	})

	r.Get("/*", s.generalHandler)
	return r
}
