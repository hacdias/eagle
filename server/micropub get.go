package server

import (
	"log"
	"net/http"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
)

func getMicropubHandler(s *services.Services, c *config.Config) http.HandlerFunc {
	config := map[string]interface{}{
		"syndicate-to": map[string]interface{}{
			"uid":  "twitter",
			"name": "Twitter",
		},
	}

	sourceHandler := micropubSource(s, c)

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("q") {
		case "source":
			sourceHandler(w, r)
		case "config", "syndicate-to":
			serveJSON(w, http.StatusOK, config)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func micropubSource(s *services.Services, c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseURL(c, r.URL.Query().Get("url"))
		if err != nil {
			log.Printf("micropub: cannot parse url: %s", err)
			serveError(w, http.StatusBadRequest, err)
			return
		}

		post, err := s.Hugo.GetEntry(id)
		if err != nil {
			log.Printf("micropub: cannot get hugo entry: %s", err)
			serveError(w, http.StatusBadRequest, err)
			return
		}

		entry := map[string]interface{}{
			"type":       []string{"h-entry"},
			"properties": post.Metadata["properties"],
		}

		if title, ok := post.Metadata.StringIf("title"); ok {
			entry["properties"].(map[string]interface{})["name"] = []string{title}
		}

		if tags, ok := post.Metadata.StringsIf("tags"); ok {
			entry["properties"].(map[string]interface{})["category"] = tags
		}

		if date, ok := post.Metadata.StringIf("date"); ok {
			entry["properties"].(map[string]interface{})["published"] = []string{date}
		}

		if post.Content != "" {
			entry["properties"].(map[string]interface{})["content"] = []string{post.Content}
		}

		serveJSON(w, http.StatusOK, entry)
	}
}
