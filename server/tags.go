package server

import (
	"net/http"

	"github.com/hacdias/eagle/v2/eagle"
)

func (s *Server) tagsGet(w http.ResponseWriter, r *http.Request) {
	tags, err := s.Tags()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	entry := s.getEntryOrEmpty(r.URL.Path)
	if entry.Title == "" {
		entry.Title = "Tags"
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: entry,
		Data: listingPage{
			Terms: tags,
		},
		IsListing: true,
	}, []string{eagle.TemplateTags})
}
