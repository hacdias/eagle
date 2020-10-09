package server

import (
	"errors"
	"strings"
)

func (s *Server) micropubParseURL(url string) (string, error) {
	if url == "" {
		return "", errors.New("url must be set")
	}

	if !strings.HasPrefix(url, s.c.Domain) {
		return "", errors.New("invalid request")
	}

	return strings.Replace(url, s.c.Domain, "", 1), nil
}
