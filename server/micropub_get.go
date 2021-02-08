package server

/*
func (s *Server) micropubSource(w http.ResponseWriter, r *http.Request) {
	s.Debug("micropub: source request received")
	id, err := s.micropubParseURL(r.URL.Query().Get("url"))
	if err != nil {
		s.Errorf("micropub: cannot parse url: %s", err)
		s.serveError(w, http.StatusBadRequest, err)
		return
	}

	post, err := s.Hugo.GetEntry(id)
	if err != nil {
		if os.IsNotExist(err) {
			s.Errorf("micropub: post not found: %s", err)
			s.serveError(w, http.StatusNotFound, fmt.Errorf("post not found: %s", id))
		} else {
			s.Errorf("micropub: cannot get hugo entry: %s", err)
			s.serveError(w, http.StatusBadRequest, err)
		}
		return
	}

	entry := map[string]interface{}{
		"type": []string{"h-entry"},
	}

	props := post.Metadata["properties"].(map[string][]interface{})

	if title, ok := post.Metadata.StringIf("title"); ok {
		props["name"] = []interface{}{title}
	}

	if tags, ok := post.Metadata.StringsIf("tags"); ok {
		props["category"] = []interface{}{}
		for _, tag := range tags {
			props["category"] = append(props["category"], tag)
		}
	}

	if date, ok := post.Metadata.StringIf("lastmod"); ok {
		props["published"] = []interface{}{date}
	} else if date, ok := post.Metadata.StringIf("date"); ok {
		props["published"] = []interface{}{date}
	}

	if post.Content != "" {
		props["content"] = []interface{}{post.Content}
	}

	entry["properties"] = props
	s.serveJSON(w, http.StatusOK, entry)
	s.Debug("micropub: source request ok")
}
*/
