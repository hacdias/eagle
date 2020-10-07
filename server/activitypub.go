package server

import (
	"net/http"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
)

func activityPubInboxHandler(s *services.Services, c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO(activitypub): implement inbox
		w.WriteHeader(http.StatusNotImplemented)
	}
}
