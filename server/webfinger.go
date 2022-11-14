package server

import (
	"fmt"
	"net/http"
	urlpkg "net/url"
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

func (s *Server) webfingerGet(w http.ResponseWriter, r *http.Request) {
	url, _ := urlpkg.Parse(s.c.Server.BaseURL)

	wf := &webfinger{
		Subject: fmt.Sprintf("acct:%s@%s", s.c.User.Username, url.Host),
		Aliases: []string{
			s.c.ID(),
		},
		Links: []link{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: s.c.ID(),
			},
		},
	}

	s.serveJSON(w, http.StatusOK, wf)
}
