package server

import (
	"net/http"

	"github.com/hacdias/eagle/v2/eagle"
)

func (s *Server) tagsGet(w http.ResponseWriter, r *http.Request) {
	tags, err := s.GetTags()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	ee := s.getListingEntryOrEmpty(r.URL.Path)
	if ee.Title == "" {
		ee.Title = "Tags"
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: ee,
		Data: listingPage{
			Terms: tags,
		},
	}, []string{eagle.TemplateTags})
}
