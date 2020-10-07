package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/micropub"
	"github.com/hacdias/eagle/services"
)

func parseURL(c *config.Config, url string) (string, error) {
	if url == "" {
		return "", errors.New("url must be set")
	}

	if !strings.HasPrefix(url, c.Domain) {
		return "", errors.New("invalid request")
	}

	return strings.Replace(url, c.Domain, "", 1), nil
}

func micropubSource(s *services.Services, c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseURL(c, r.URL.Query().Get("url"))
		if err != nil {
			serveJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		post, err := s.Hugo.GetEntry(id)
		if err != nil {
			serveJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
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

func postMicropubHandler(s *services.Services, c *config.Config) http.HandlerFunc {
	create := micropubCreate(s, c)
	update := micropubUpdate(s, c)
	remove := micropubRemove(s, c)
	unremove := micropubUnremove(s, c)

	return func(w http.ResponseWriter, r *http.Request) {
		mr, err := micropub.ParseRequest(r)
		if err != nil {
			serveError(w, http.StatusBadRequest, err)
			return
		}

		var code int

		switch mr.Action {
		case micropub.ActionCreate:
			code, err = create(w, r, mr)
		case micropub.ActionUpdate:
			code, err = update(w, r, mr)
		case micropub.ActionDelete:
			code, err = remove(w, r, mr)
		case micropub.ActionUndelete:
			code, err = unremove(w, r, mr)
		default:
			code, err = http.StatusBadRequest, errors.New("invalid action")
		}

		if code >= 200 && code < 400 {
			w.WriteHeader(code)
		} else if code >= 400 {
			serveError(w, code, err)
		}

		err = s.Hugo.Build(mr.Action == micropub.ActionDelete)
		if err != nil {
			s.Notify.Error(err)
		}
	}
}
