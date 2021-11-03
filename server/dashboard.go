package server

// TODO:
// - s.Sync: Sync was successfull! ⚡️
// - s.RebuildIndex: "Search index rebuilt! 🔎"
// - resend webmentions

// func recentlyTemplate() (*eagle.Entry, string) {
// 	t := time.Now()
// 	month := t.Format("January")

// 	entry := &eagle.Entry{
// 		Content: "How was last month?",
// 		Frontmatter: eagle.Frontmatter{
// 			Draft:     true,
// 			Title:     fmt.Sprintf("Recently in %s '%s", month, t.Format("06")),
// 			Published: t,
// 			Properties: map[string]interface{}{
// 				"categories": []string{"recently"},
// 			},
// 		},
// 	}

// 	id := fmt.Sprintf("/articles/%s-%s/", strings.ToLower(month), t.Format("2006"))
// 	return entry, id
// }

// func defaultTemplate() (*eagle.Entry, string) {
// 	t := time.Now()

// 	entry := &eagle.Entry{
// 		Content: "Lorem ipsum...",
// 		Frontmatter: eagle.Frontmatter{
// 			Draft:     true,
// 			Published: t,
// 			Properties: map[string]interface{}{
// 				"categories": []string{"example"},
// 			},
// 		},
// 	}

// 	id := fmt.Sprintf("micro/%s/SLUG", t.Format("2006/01"))
// 	return entry, id
// }

// func (s *Server) newGetHandler(w http.ResponseWriter, r *http.Request) {
// 	template := r.URL.Query().Get("template")

// 	var (
// 		entry *eagle.Entry
// 		id    string
// 	)

// 	switch template {
// 	case "recently":
// 		entry, id = recentlyTemplate()
// 	default:
// 		entry, id = defaultTemplate()
// 	}

// 	reply := sanitizeReplyURL(r.URL.Query().Get("reply"))
// 	if reply != "" {
// 		// var err error
// 		// entry.Metadata.ReplyTo, err = s.GetXRay(reply)
// 		// if err != nil {
// 		// 	s.dashboardError(w, r, err)
// 		// 	return
// 		// }
// 	}

// 	str, err := entry.String()
// 	if err != nil {
// 		s.dashboardError(w, r, err)
// 		return
// 	}

// 	s.renderDashboard(w, "new", &dashboardData{
// 		Content: str,
// 		ID:      id,
// 	})
// }

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

// func (s *Server) newPostHandler(w http.ResponseWriter, r *http.Request) {
// 	err := r.ParseForm()
// 	if err != nil {
// 		s.dashboardError(w, r, err)
// 		return
// 	}

// 	content := r.FormValue("content")
// 	twitter := r.FormValue("twitter") == "on"

// 	id, err := sanitizeID(r.FormValue("id"))
// 	if err != nil {
// 		s.dashboardError(w, r, err)
// 		return
// 	}

// 	if id == "" {
// 		s.dashboardError(w, r, errors.New("no ID provided"))
// 		return
// 	}

// 	if id == r.FormValue("defaultid") {
// 		s.dashboardError(w, r, errors.New("cannot use default ID"))
// 		return
// 	}

// 	entry, err := s.ParseEntry(id, content)
// 	if err != nil {
// 		s.dashboardError(w, r, err)
// 		return
// 	}

// 	err = s.newEditPostSaver(entry, false, twitter)
// 	if err != nil {
// 		s.dashboardError(w, r, err)
// 		return
// 	}

// 	if entry.Draft {
// 		s.redirectWithStatus(w, entry.ID+" updated successfullyl! ⚡️")
// 		return
// 	}

// 	http.Redirect(w, r, entry.Permalink, http.StatusTemporaryRedirect)
// }
