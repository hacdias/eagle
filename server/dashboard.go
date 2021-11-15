package server

import (
	"net/http"

	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/entry"
)

func (s *Server) dashboardGet(w http.ResponseWriter, r *http.Request) {
	s.serveHTML(w, r, &eagle.RenderData{
		Entry: &entry.Entry{
			Frontmatter: entry.Frontmatter{
				Title: "Dashboard",
			},
		},
		NoIndex: true,
	}, []string{eagle.TemplateDashboard})
}

func (s *Server) dashboardPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	actions := r.Form["action"]

	for _, action := range actions {
		switch action {
		case "clear-cache":
			s.ResetCache()
		case "sync-storage":
			go s.SyncStorage()
		case "update-blogroll":
			err = s.UpdateBlogroll()
		}
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
