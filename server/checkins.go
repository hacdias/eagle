package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/renderer"
)

func (s *Server) newCheckinGet(w http.ResponseWriter, r *http.Request) {
	s.serveHTML(w, r, &renderer.RenderData{
		Entry: &eagle.Entry{
			FrontMatter: eagle.FrontMatter{
				Title: "New Checkin",
			},
		},
		NoIndex: true,
	}, []string{renderer.TemplateNewCheckin})
}

func (s *Server) newCheckinPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	name := r.FormValue("name")
	geouri := r.FormValue("location")
	if name == "" || geouri == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("name or location is empty"))
		return
	}

	var (
		date time.Time
		err  error
	)

	if dateStr := r.FormValue("date"); dateStr != "" {
		date, err = dateparse.ParseStrict(dateStr)
	} else {
		date = time.Now()
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	location, err := s.maze.ReverseGeoURI(s.c.Site.Language, geouri)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	c := &eagle.Checkin{
		Date:      date,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
		Name:      name,
		Locality:  location.Locality,
		Region:    location.Region,
		Country:   location.Country,
	}

	err = s.fs.SaveCheckin(c)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/checkins", http.StatusSeeOther)
}
