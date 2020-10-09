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

		if date, ok := post.Metadata.StringIf("date"); ok {
			props["published"] = []interface{}{date}
		}

		if post.Content != "" {
			props["content"] = []interface{}{post.Content}
		}

		entry["properties"] = props
		serveJSON(w, http.StatusOK, entry)
	}
}
