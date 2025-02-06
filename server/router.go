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

	if s.c.Tor {
		r.Use(s.withOnionHeader)
	}

	if s.c.Comments.Redirect != "" {
		r.Post(commentsPath, s.commentsPost)
	}

	if s.c.WebhookSecret != "" {
		r.Post(webhookPath, s.webhookPost)
	}

	if s.c.Webmentions.Secret != "" {
		r.Post(webmentionPath, s.webmentionPost)
	}

	if s.meilisearch != nil {
		r.Get(searchPath, s.searchGet)
	}

	if s.c.Site.Params.Author.Handle != "" {
		r.Get(wellKnownWebFingerPath, s.makeWellKnownWebFingerGet())
	}

	// TODO: make this customizable. Plugin?
	r.Get(wellKnownAvatarPath, s.wellKnownAvatarPath)

	// IndieAuth Server (Part I)
	r.Get(wellKnownOAuthServer, s.indieauthGet)
	r.Post(authPath, s.authPost)
	r.Post(tokenPath, s.tokenPost)
	r.Post(tokenVerifyPath, s.tokenVerifyPost)

	// Panel Assets
	r.Get("/panel/assets*", http.StripPrefix("/panel", http.FileServer(http.FS(panelAssetsFS))).ServeHTTP)

	// Panel Pages
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(s.jwtAuth))
		r.Use(s.withLoggedIn)

		// Login
		r.Get(loginPath, s.loginGet)
		r.Post(loginPath, s.loginPost)
		r.Get(logoutPath, s.logoutGet)

		// Admin only pages.
		r.Group(func(r chi.Router) {
			r.Use(s.mustLoggedIn)

			// IndieAuth Server (Part II)
			r.Get(authPath, s.authGet)
			r.Post(authAcceptPath, s.authAcceptPost)

			// Panel
			r.Get(panelPath, s.panelGet)
			r.Post(panelPath, s.panelPost)
			r.Get(panelMentionsPtah, s.panelMentionsGet)
			r.Post(panelMentionsPtah, s.panelMentionsPost)
			r.Get(panelTokensPath, s.panelTokensGet)
			r.Post(panelTokensPath, s.panelTokensPost)
			r.Get(panelBrowsePath+"*", s.panelBrowserGet)
			r.Post(panelBrowsePath+"*", s.panelBrowserPost)
			r.Get(panelEditPath+"*", s.panelEditGet)
			r.Post(panelEditPath+"*", s.panelEditPost)
		})
	})

	// IndieAuth-protected Pages
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(s.jwtAuth))
		r.Use(s.mustIndieAuth)

		// IndieAuth Server (Part III)
		r.Get(tokenPath, s.tokenGet) // Backwards compatible token verification endpoint
		r.Get(userInfoPath, s.userInfoGet)

		// Micropub
		if s.c.Micropub != nil {
			r.Handle(micropubPath, s.makeMicropub())
			if s.media != nil {
				r.Handle(micropubMediaPath, s.makeMicropubMedia())
			}
		}
	})

	// Do not server Hugo's 404.html as 200 OK.
	r.Get("/404.html", func(w http.ResponseWriter, r *http.Request) {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	})

	// Plugin Pages
	utilities := &PluginWebUtilities{s: s}
	for _, plugin := range s.plugins {
		handlerPlugin, ok := plugin.(HandlerPlugin)
		if !ok {
			continue
		}

		r.HandleFunc(handlerPlugin.HandlerRoute(), func(w http.ResponseWriter, r *http.Request) {
			handlerPlugin.Handler(w, r, utilities)
		})
	}

	// Everything Bagel 🥯
	r.Get("/*", s.everythingBagelHandler)
	return r
}

func (s *Server) everythingBagelHandler(w http.ResponseWriter, r *http.Request) {
	if url, ok := s.redirects[r.URL.Path]; ok {
		http.Redirect(w, r, url, http.StatusMovedPermanently)
		return
	}

	if _, ok := s.gone[r.URL.Path]; ok {
		s.serveErrorHTML(w, r, http.StatusGone, nil)
		return
	}

	nfw := &notFoundResponseWriter{ResponseWriter: w}
	s.staticFs.ServeHTTP(nfw, r)

	if nfw.status == http.StatusNotFound {
		e, err := s.core.GetEntry(r.URL.Path)
		if err == nil && e.Deleted() {
			s.serveErrorHTML(w, r, http.StatusGone, nil)
		} else {
			s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		}
	}
}

// notFoundResponseWriter wraps a Response Writer to capture 404 requests.
// In case it is a 404 request, then we do not write the body.
type notFoundResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *notFoundResponseWriter) WriteHeader(status int) {
	w.status = status
	if status != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *notFoundResponseWriter) Write(p []byte) (int, error) {
	if w.status != http.StatusNotFound {
		return w.ResponseWriter.Write(p)
	}
	// Lie that we successfully written it
	return len(p), nil
}
