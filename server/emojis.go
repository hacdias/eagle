package server

import (
	"net/http"

	"github.com/hacdias/eagle/v4/eagle"
)

func (s *Server) emojisGet(w http.ResponseWriter, r *http.Request) {
	emojis, err := s.GetEmojis()
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	s.serveHTML(w, r, &eagle.RenderData{
		Entry: s.getListingEntryOrEmpty(r.URL.Path, "Emojis"),
		Data: listingPage{
			Terms: emojis,
		},
	}, []string{eagle.TemplateEmojis})
}
