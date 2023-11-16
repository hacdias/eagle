package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"go.hacdias.com/eagle/log"
)

func (s *Server) makeRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(s.withRecoverer)
	r.Use(log.WithZap)
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
	r.Get(wellKnownAvatarPath, s.wellKnownAvatarPath)
	r.Post(guestbookPath, s.guestbookPost)
	if s.meilisearch != nil {
		// TODO: ensure /search /search/
		r.Get(searchPath, s.searchGet)
	}

	utilities := &PluginWebUtilities{s: s}
	for _, plugin := range s.plugins {
		route, handler := plugin.GetWebHandler(utilities)
		if route != "" {
			r.HandleFunc(route, handler)
		}
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

		r.Get(panelPath, s.panelGet)
		r.Post(panelPath, s.panelPost)

		r.Get(panelGuestbookPath, s.panelGuestbookGet)
		r.Post(panelGuestbookPath, s.panelGuestbookPost)
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
