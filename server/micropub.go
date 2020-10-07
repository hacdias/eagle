package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/micropub"
)

type micropubHandlerFunc func(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error)

func parseURL(c *config.Config, url string) (string, error) {
	if url == "" {
		return "", errors.New("url must be set")
	}

	if !strings.HasPrefix(url, c.Domain) {
		return "", errors.New("invalid request")
	}

	return strings.Replace(url, c.Domain, "", 1), nil
}
