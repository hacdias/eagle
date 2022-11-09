package server

import (
	"fmt"
	"net/http"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/renderer"
	"github.com/hacdias/indieauth/v3"
)

func (s *Server) dashboardGet(w http.ResponseWriter, r *http.Request) {
	s.serveDashboard(w, r, map[string]interface{}{})
}

func (s *Server) dashboardPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	actions := r.Form["action"]

	data := map[string]interface{}{}

	for _, action := range actions {
		switch action {
		case "clear-cache":
			s.cache.Clear()
			data["Success"] = true
		case "sync-storage":
			go s.SyncStorage()
			data["Success"] = true
		case "update-blogroll":
			// wip: use dashboardActions
			// err = s.UpdateMinifluxBlogroll()
			data["Success"] = true
		case "update-reads-statistics":
			// wip: use dashboardActions
			// err = s.UpdateReadsSummary()
			data["Success"] = true
		case "update-watches-statistics":
			// wip: use dashboardActions
			// err = s.UpdateWatchesSummary()
			data["Success"] = true
		case "token":
			clientID := r.Form.Get("client_id")
			scope := r.Form.Get("scope")
			expiry, err := handleExpiry(r.Form.Get("expiry"))
			if err != nil {
				s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("expiry param is invalid: %w", err))
			}

			if err := indieauth.IsValidClientIdentifier(clientID); err != nil {
				s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("invalid client_id: %w", err))
				return
			}

			signed, err := s.generateToken(clientID, scope, expiry)
			if err == nil {
				data["Token"] = signed
			}
		}
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveDashboard(w, r, data)
}

func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request, data interface{}) {
	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &entry.Entry{
			FrontMatter: entry.FrontMatter{
				Title: "Dashboard",
			},
		},
		Data:    data,
		NoIndex: true,
	}, []string{renderer.TemplateDashboard})
}
