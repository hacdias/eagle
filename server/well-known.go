package server

import (
	"fmt"
	"net/http"
	urlpkg "net/url"
	"strconv"

	"github.com/hacdias/eagle/core"
)

// TODO: consider moving this to simple rewrites in Caddy, if possible, of course.
// That would be nice as I could easily ensure that the static files on disk would
// correspond with the response for the requests we want.
//
// In addition, I wouldn't need to have the "overhead" for the /.well-known/links
// of maintaining it in memory and refreshing the values once in a while.

const (
	wellKnownWebFingerPath = "/.well-known/webfinger"
	wellKnownLinksPath     = "/.well-known/links"
	wellKnownAvatarPath    = "/.well-known/avatar"
)

func (s *Server) initWebFinger() {
	url, _ := urlpkg.Parse(s.c.BaseURL)

	s.webFinger = &core.WebFinger{
		Subject: fmt.Sprintf("acct:%s@%s", s.c.User.Username, url.Host),
		Aliases: []string{
			s.c.BaseURL,
		},
		Links: []core.WebFingerLink{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: s.c.BaseURL,
			},
		},
	}
}

func (s *Server) wellKnownWebFingerGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("resource") == s.webFinger.Subject {
		s.serveJSON(w, http.StatusOK, s.webFinger)
	} else {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	}
}

func (s *Server) wellKnownLinksGet(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		s.serveJSON(w, http.StatusOK, s.links)
	} else if v, ok := s.linksMap[domain]; ok {
		s.serveJSON(w, http.StatusOK, v)
	} else {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	}
}

func (s *Server) wellKnownAvatarPath(w http.ResponseWriter, r *http.Request) {
	size := r.URL.Query().Get("size")
	if size == "" {
		size = "512"
	}

	_, err := strconv.Atoi(size)
	if err != nil {
		// Invalid argument!
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	// Rewrite request to correct avatar path.
	r.URL.Path = "/avatar/" + size + ".jpg"
	s.generalHandler(w, r)
}
