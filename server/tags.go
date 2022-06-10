package server

import (
	"net/http"

	"github.com/hacdias/eagle/v4/eagle"
)

func (s *Server) tagsGet(w http.ResponseWriter, r *http.Request) {
	tags, err := s.GetTags()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: s.getListingEntryOrEmpty(r.URL.Path, "Tags"),
		Data: listingPage{
			Terms: tags,
		},
	}, []string{eagle.TemplateTags})
}
