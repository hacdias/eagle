package server

import (
	"fmt"
	urlpkg "net/url"
	"path"
	"strings"

	"github.com/hacdias/eagle/eagle"
)

func (s *Server) goSyndicate(entry *eagle.Entry) {
	if s.Twitter == nil {
		return
	}

	url, err := s.Twitter.Syndicate(entry)
	if err != nil {
		s.NotifyError(fmt.Errorf("failed to syndicate: %w", err))
		return
	}

	entry.Metadata.Syndication = append(entry.Metadata.Syndication, url)
	err = s.SaveEntry(entry)
	if err != nil {
		s.NotifyError(fmt.Errorf("failed to save entry: %w", err))
		return
	}

	err = s.Build(false)
	if err != nil {
		s.NotifyError(fmt.Errorf("failed to build: %w", err))
	}
}

func (s *Server) goWebmentions(entry *eagle.Entry) {
	err := s.SendWebmentions(entry)
	if err != nil {
		s.NotifyError(fmt.Errorf("webmentions: %w", err))
	}
}

func sanitizeReplyURL(replyUrl string) string {
	if strings.HasPrefix(replyUrl, "https://twitter.com") && strings.Contains(replyUrl, "/status/") {
		url, err := urlpkg.Parse(replyUrl)
		if err != nil {
			return replyUrl
		}

		url.RawQuery = ""
		url.Fragment = ""

		return url.String()
	}

	return replyUrl
}

func sanitizeID(id string) (string, error) {
	if id != "" {
		url, err := urlpkg.Parse(id)
		if err != nil {
			return "", err
		}
		id = path.Clean(url.Path)
	}
	return id, nil
}
