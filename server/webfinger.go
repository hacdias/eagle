package server

import (
	"fmt"
	"net/http"
	urlpkg "net/url"

	"github.com/hacdias/eagle/core"
)

const (
	webFingerPath = "/.well-known/webfinger"
)

func (s *Server) initWebFinger() {
	url, _ := urlpkg.Parse(s.c.Server.BaseURL)

	s.webFinger = &core.WebFinger{
		Subject: fmt.Sprintf("acct:%s@%s", s.c.User.Username, url.Host),
		Aliases: []string{
			s.c.Server.BaseURL,
		},
		Links: []core.WebFingerLink{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: s.c.Server.BaseURL,
			},
		},
	}
}

func (s *Server) webFingerGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("resource") == s.webFinger.Subject {
		s.serveJSON(w, http.StatusOK, s.webFinger)
	} else {
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	}
}
