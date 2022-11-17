package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/hacdias/eagle/renderer"
)

const (
	feedPath  = ".{feed:rss|atom|json}"
	yearPath  = `/{year:(x|\d\d\d\d)}`
	monthPath = yearPath + `/{month:(x|\d\d)}`
	dayPath   = monthPath + `/{day:(x|\d\d)}`
)

func (s *Server) makeRouter() http.Handler {
	r := chi.NewRouter()

	if s.c.Server.Logging || s.c.Development {
		r.Use(middleware.Logger)
	}

	r.Use(middleware.RedirectSlashes)
	r.Use(cleanPath)
	r.Use(middleware.GetHead)

	r.Use(s.recoverer)
	r.Use(s.securityHeaders)

	r.Use(jwtauth.Verifier(s.jwtAuth))
	r.Use(s.withLoggedIn)

	if s.c.Server.Tor != nil {
		r.Use(s.onionHeader)
		r.Get("/onion", s.onionRedirGet)
	}

	if s.c.Server.WebhookSecret != "" {
		r.Post("/webhook", s.webhookPost)
	}

	if s.c.Webmentions.Secret != "" {
		r.Post("/webmention", s.webmentionPost)
	}

	if s.ap != nil {
		r.Post("/activitypub/inbox", s.activityPubInboxPost)
		r.Get("/activitypub/outbox", s.activityPubOutboxGet)
	}

	r.Get("/search", s.searchGet)
	r.Get(renderer.AssetsBaseURL+"*", s.serveAssets)
	r.Get("/.well-known/webfinger", s.webfingerGet)
	r.Get("/on-this-day", s.onThisDayGet)

	// IndieAuth Server
	r.Get("/.well-known/oauth-authorization-server", s.indieauthGet)
	r.Get("/auth", s.authGet)
	r.Post("/auth/accept", s.authAcceptPost)
	r.Post("/auth", s.authPost)
	r.Post("/token", s.tokenPost)
	r.Post("/token/verify", s.tokenVerifyPost)

	// Tiles API
	if s.c.Server.TilesSource != "" {
		r.Get("/tiles/{s}/{z}/{x}/{y}", s.tilesGet)
		r.Get("/tiles/{s}/{z}/{x}/{y}@{r}", s.tilesGet)
	}

	// IndieAuth Client
	r.Get("/login", s.loginGet)
	r.Post("/login", s.loginPost)
	r.Get("/login/callback", s.loginCallbackGet)
	r.Get("/logout", s.logoutGet)

	r.Group(func(r chi.Router) {
		r.Use(s.mustIndieAuth)

		r.Get("/micropub", s.micropubGet)
		r.Post("/micropub", s.micropubPost)
		r.Post("/micropub/media", s.micropubMediaPost)

		// IndieAuth Server
		r.Get("/token", s.tokenGet) // Backwards compatible token verification endpoint
		r.Get("/userinfo", s.userInfoGet)
	})

	// Admin only pages.
	r.Group(func(r chi.Router) {
		r.Use(s.mustLoggedIn)
		r.Use(s.mustAdmin)

		r.Get("/new", s.newGet)
		r.Post("/new", s.newPost)

		r.Get("/edit*", s.editGet)
		r.Post("/edit*", s.editPost)

		r.Get("/dashboard", s.dashboardGet)
		r.Post("/dashboard", s.dashboardPost)

		if s.ap != nil {
			r.Post("/dashboard/activitypub", s.activityPubHookPost)
		}

		r.Get("/deleted", s.deletedGet)
		r.Get("/drafts", s.draftsGet)
		r.Get("/unlisted", s.unlistedGet)

		r.Get("/mention-toggle*", s.mentionToggleGet)
	})

	// Logged-in only pages.
	r.Group(func(r chi.Router) {
		r.Use(s.mustLoggedIn)

		r.Get("/private", s.privateGet)
	})

	// Listing HTML pages. Cached.
	r.Group(func(r chi.Router) {
		r.Use(s.withCache)

		r.Get("/", s.indexGet)
		r.Get("/all", s.allGet)
		r.Get(yearPath, s.dateGet)
		r.Get(monthPath, s.dateGet)
		r.Get(dayPath, s.dateGet)

		for _, section := range s.c.Site.Sections {
			if section != s.c.Site.IndexSection {
				r.Get("/"+section, s.sectionGet(section))
			}
		}

		for id, taxonomy := range s.c.Site.Taxonomies {
			r.Get("/"+id, s.taxonomyGet(id, taxonomy))
			r.Get("/"+id+"/{term}", s.taxonomyTermGet(id, taxonomy))
		}
	})

	// Listing JSON, XML and ATOM feeds. Not cached.
	r.Get("/"+feedPath, s.indexGet)
	r.Get("/all"+feedPath, s.allGet)
	r.Get(yearPath+feedPath, s.dateGet)
	r.Get(monthPath+feedPath, s.dateGet)
	r.Get(dayPath+feedPath, s.dateGet)

	for _, section := range s.c.Site.Sections {
		if section != s.c.Site.IndexSection {
			r.Get("/"+section+feedPath, s.sectionGet(section))
		}
	}

	for id, taxonomy := range s.c.Site.Taxonomies {
		r.Get("/"+id+"/{term}"+feedPath, s.taxonomyTermGet(id, taxonomy))
	}

	// Everything that was not matched so far.
	r.Group(func(r chi.Router) {
		r.Use(s.withRedirects)
		r.Use(s.withStaticFiles)
		r.Use(s.withCache)

		r.Get("/*", s.entryGet)
	})

	return r
}
