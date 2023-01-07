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
		t   time.Time
		err error
	)

	if dateStr := r.FormValue("date"); dateStr != "" {
		t, err = dateparse.ParseStrict(dateStr)
	} else {
		t = time.Now()
	}

	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	location, err := s.maze.ReverseGeoURI(s.c.Site.Language, geouri)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, err)
		return
	}

	location.Name = name
	c := &eagle.Checkin{
		Location: *location,
		Date:     t,
	}

	err = s.fs.SaveCheckin(c)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(w, r, "/checkins", http.StatusSeeOther)
}
