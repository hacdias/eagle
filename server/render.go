package server

import (
	"net/http"

	"github.com/hacdias/eagle/v2/eagle"
)

func (s *Server) render(w http.ResponseWriter, data *eagle.RenderData, tpls []string) {
	data.TorUsed = false
	data.OnionAddress = ""
	data.LoggedIn = false

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err := s.Render(w, data, tpls)
	if err != nil {
		panic(err)
	}
}
