package server

// TODO(v2):
// - s.Sync: Sync was successfull! ⚡️
// - s.RebuildIndex: "Search index rebuilt! 🔎"
// - Blogroll?
// - resend webmentions

// func (s *Server) blogrollGetHandler(w http.ResponseWriter, r *http.Request) {
// 	if s.Miniflux == nil {
// 		s.dashboardError(w, r, errors.New("miniflux integration is disabled"))
// 		return
// 	}

// 	feeds, err := s.Miniflux.Fetch()
// 	if err != nil {
// 		s.dashboardError(w, r, err)
// 		return
// 	}

// 	data, err := json.MarshalIndent(feeds, "", "  ")
// 	if err != nil {
// 		s.dashboardError(w, r, err)
// 		return
// 	}

// 	s.renderDashboard(w, "gedit", &dashboardData{
// 		ID:      "data/blogroll.json",
// 		Content: string(data),
// 	})
// }
