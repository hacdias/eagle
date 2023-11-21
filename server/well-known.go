package server

import (
	"fmt"
	"net/http"
	urlpkg "net/url"
	"strconv"
)

const (
	wellKnownWebFingerPath = "/.well-known/webfinger"
	wellKnownAvatarPath    = "/.well-known/avatar"
)

type webFinger struct {
	Subject string          `json:"subject"`
	Aliases []string        `json:"aliases,omitempty"`
	Links   []webFingerLink `json:"links,omitempty"`
}

type webFingerLink struct {
	Href     string `json:"href"`
	Rel      string `json:"rel,omitempty"`
	Type     string `json:"type,omitempty"`
	Template string `json:"template,omitempty"`
}

func (s *Server) makeWellKnownWebFingerGet() http.HandlerFunc {
	url, _ := urlpkg.Parse(s.c.BaseURL)

	webFinger := &webFinger{
		Subject: fmt.Sprintf("acct:%s@%s", s.c.Site.Params.Author.Handle, url.Host),
		Aliases: []string{
			s.c.BaseURL,
		},
		Links: []webFingerLink{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: s.c.BaseURL,
			},
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("resource") == webFinger.Subject {
			s.serveJSON(w, http.StatusOK, webFinger)
		} else {
			s.serveErrorHTML(w, r, http.StatusNotFound, nil)
		}
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
	s.everythingBagelHandler(w, r)
}
