package server

import (
	"fmt"
	"net/http"
	urlpkg "net/url"

	"github.com/hacdias/eagle/pkg/contenttype"
)

type webfinger struct {
	Subject string   `json:"subject"`
	Aliases []string `json:"aliases,omitempty"`
	Links   []link   `json:"links,omitempty"`
}

type link struct {
	Href string `json:"href"`
	Rel  string `json:"rel,omitempty"`
	Type string `json:"type,omitempty"`
}

func (s *Server) initWebfinger() {
	url, _ := urlpkg.Parse(s.c.Server.BaseURL)

	s.webfinger = &webfinger{
		Subject: fmt.Sprintf("acct:%s@%s", s.c.User.Username, url.Host),
		Aliases: []string{
			s.c.Server.BaseURL,
		},
		Links: []link{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: s.c.Server.BaseURL,
			},
		},
	}

	if s.ap != nil {
		s.webfinger.Links = append(s.webfinger.Links, link{
			Href: s.c.Server.BaseURL,
			Rel:  "self",
			Type: contenttype.AS,
		})
	}
}

func (s *Server) webfingerGet(w http.ResponseWriter, r *http.Request) {
	s.serveJSON(w, http.StatusOK, s.webfinger)
}
