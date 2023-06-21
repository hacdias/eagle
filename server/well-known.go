package server

import (
	"fmt"
	"net/http"
	urlpkg "net/url"

	"github.com/hacdias/eagle/core"
)

const (
	wellKnownWebFingerPath = "/.well-known/webfinger"
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

const (
	wellKnownLinksPath = "/.well-known/links"
)

func (s *Server) wellKnownLinksGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != wellKnownLinksPath {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		s.serveJSON(w, http.StatusOK, s.links)
	} else if v, ok := s.linksMap[domain]; ok {
		s.serveJSON(w, http.StatusOK, v)
	} else {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	}
}
