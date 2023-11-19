package server

import (
	"net/http"
	"strings"

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
	r.Post(commentsPath, s.commentsPost)

	// GitHub WebHook
	if s.c.WebhookSecret != "" {
		r.Post(webhookPath, s.webhookPost)
	}

	// Webmentions Handler
	if s.c.Webmentions.Secret != "" {
		r.Post(webmentionPath, s.webmentionPost)
	}

	// WebFinger if handle is defined
	if s.c.Site.Params.Author.Handle != "" {
		r.Get(wellKnownWebFingerPath, s.makeWellKnownWebFingerGet())
	}

	// TODO: make this customizable. Plugin?
	r.Get(wellKnownAvatarPath, s.wellKnownAvatarPath)

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
	r.Post(tokenPath, s.tokenPost)
	r.Post(tokenVerifyPath, s.tokenVerifyPost)

	// Admin only pages.
	r.Group(func(r chi.Router) {
		r.Use(s.mustLoggedIn)

		// IndieAuth Server (Part II)
		r.Get(authPath+"/", s.authGet)
		r.Post(authAcceptPath, s.authAcceptPost)

		// Panel
		r.Get(panelPath, s.panelGet)
		r.Post(panelPath, s.panelPost)

		r.Get(panelMentionsPtah, s.panelMentionsGet)
		r.Post(panelMentionsPtah, s.panelMentionsPost)

		r.Get(panelTokensPath, s.panelTokensGet)
		r.Post(panelTokensPath, s.panelTokensPost)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.mustIndieAuth)

		// IndieAuth Server (Part III)
		r.Get(tokenPath, s.tokenGet) // Backwards compatible token verification endpoint
		r.Get(userInfoPath, s.userInfoGet)
	})

	// Ensure this routes are redirected to the slash to avoid 404.
	for _, route := range []string{
		searchPath, authPath, loginPath, logoutPath,
		panelPath, panelMentionsPtah, panelTokensPath,
	} {
		r.Get(strings.TrimSuffix(route, "/"), func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path += "/"
			http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
		})
	}

	// Protect special templating page.
	r.Get("/_eagle", s.serveNotFound)
	r.Get("/_eagle/", s.serveNotFound)

	// Plugins that mount routes.
	utilities := &PluginWebUtilities{s: s}
	for _, plugin := range s.plugins {
		route, handler := plugin.GetWebHandler(utilities)
		if route != "" {
			r.HandleFunc(route, handler)
		}
	}

	r.Get("/*", s.generalHandler)
	return r
}
