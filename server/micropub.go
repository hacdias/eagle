package server

import (
	"net/http"

	"github.com/hacdias/eagle/micropub"
)

func getMicropubHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("q") {
	case "source":
		url := r.URL.Query().Get("url")
		if url == "" {
			serveJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": "url must be set on source query",
			})

			return
		}

		// TODO: req.query url
		serveJSON(w, http.StatusOK, nil)
	case "config":
		serveJSON(w, http.StatusOK, map[string]interface{}{
			"config": nil, // config
		})
	case "syndicate-to":
		serveJSON(w, http.StatusOK, map[string]interface{}{
			"syndicate-to": nil, // config['syndicate-to']
		})
	default:

		w.WriteHeader(http.StatusNotFound)
	}
}

func postMicropubHandler(w http.ResponseWriter, r *http.Request) {
	req, err := micropub.ParseRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch req.Action {
	case micropub.ActionCreate:

	case micropub.ActionUpdate:

	case micropub.ActionDelete:

	case micropub.ActionUndelete:

	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Action == micropub.ActionDelete {
		// TODO: Build and Clean
	} else {
		// TODO: Build
	}

	// ERROR: notify!

	serveJSON(w, 200, req)
}
