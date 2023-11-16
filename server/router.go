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

	if s.c.Logging || s.c.Development {
		r.Use(middleware.Logger)
	}

	r.Use(withCleanPath)
	r.Use(middleware.GetHead)
	r.Use(s.withSecurityHeaders)
	r.Use(jwtauth.Verifier(s.jwtAuth))
	r.Use(s.withLoggedIn)
	r.Use(s.withAdminBar)

	// GitHub WebHook
	if s.c.WebhookSecret != "" {
		r.Post(webhookPath, s.webhookPost)
	}

	// Random
	if s.c.Site.Params.Author.Handle != "" {
		r.Get(wellKnownWebFingerPath, s.makeWellKnownWebFingerGet())
	}
	r.Get(wellKnownLinksPath, s.wellKnownLinksGet)
	r.Get(wellKnownAvatarPath, s.wellKnownAvatarPath)
	r.Post(guestbookPath, s.guestbookPost)
	if s.meilisearch != nil {
		r.Get(searchPath, s.searchGet)
	}

	// Login
	r.Get(loginPath, s.loginGet)
	r.Post(loginPath, s.loginPost)
	r.Get(logoutPath, s.logoutGet)

	// IndieAuth Server (Part I)
	r.Get(wellKnownOAuthServer, s.indieauthGet)
	r.Post(authPath, s.authPost)
	r.Post(authPath+"/", s.authPost)
	r.Post(tokenPath, s.tokenPost)
	r.Post(tokenVerifyPath, s.tokenVerifyPost)

	// Admin only pages.
	r.Group(func(r chi.Router) {
		r.Use(s.mustLoggedIn)

		// IndieAuth Server (Part II)
		r.Get(authPath+"/", s.authGet)
		r.Post(authAcceptPath, s.authAcceptPost)

		r.Get(dashboardPath, s.dashboardGet)
		r.Post(dashboardPath, s.dashboardPost)
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
