package server

import (
	"fmt"
	"net/http"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/renderer"
	"github.com/hacdias/indieauth/v3"
	"github.com/hashicorp/go-multierror"
)

type dashboardData struct {
	Actions []string
	Success bool
	Token   string
}

func (s *Server) dashboardGet(w http.ResponseWriter, r *http.Request) {
	s.serveDashboard(w, r, &dashboardData{
		Actions: s.getActions(),
	})
}

func (s *Server) dashboardPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	actions := r.Form["action"]
	token := r.Form.Get("token") == "true"
	data := &dashboardData{
		Actions: s.getActions(),
	}

	var errs *multierror.Error
	for _, actionName := range actions {
		if fn, ok := s.actions[actionName]; ok {
			errs = multierror.Append(errs, fn())
			data.Success = true
		}
	}
	if err := errs.ErrorOrNil(); err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	if token {
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
			data.Token = signed
		} else {
			s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
			return
		}
	}

	s.serveDashboard(w, r, data)
}

func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request, data *dashboardData) {
	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &eagle.Entry{
			FrontMatter: eagle.FrontMatter{
				Title: "Dashboard",
			},
		},
		Data:    data,
		NoIndex: true,
	}, []string{renderer.TemplateDashboard})
}
