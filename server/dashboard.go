package server

import (
	"errors"
	"net/http"

	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/entry"
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
			s.ResetCache()
			data["Message"] = "Success!"
		case "sync-storage":
			go s.SyncStorage()
			data["Message"] = "Success!"
		case "update-blogroll":
			err = s.UpdateBlogroll()
			data["Message"] = "Success!"
		case "update-reads-statistics":
			err = s.UpdateReadStatistics()
			data["Message"] = "Success!"
		case "token":
			clientID := r.Form.Get("client_id")
			scope := r.Form.Get("scope")
			expires := r.Form.Get("expiry") != "infinity"

			if !isValidProfileURL(clientID) {
				s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("client id is invalid"))
				return
			}

			signed, err := s.generateToken(clientID, scope, expires)
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
	s.serveHTML(w, r, &eagle.RenderData{
		Entry: &entry.Entry{
			Frontmatter: entry.Frontmatter{
				Title: "Dashboard",
			},
		},
		Data:    data,
		NoIndex: true,
	}, []string{eagle.TemplateDashboard})
}
