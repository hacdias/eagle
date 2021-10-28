package server

import (
	"bytes"
	"fmt"
	urlpkg "net/url"
	"path"
	"strings"

	"github.com/hacdias/eagle/eagle"
	"willnorris.com/go/webmention"
)

func (s *Server) goSyndicate(entry *eagle.Entry) {
	if s.e.Twitter == nil {
		return
	}

	url, err := s.e.Twitter.Syndicate(entry)
	if err != nil {
		s.e.NotifyError(fmt.Errorf("failed to syndicate: %w", err))
		return
	}

	entry.Metadata.Syndication = append(entry.Metadata.Syndication, url)
	err = s.e.SaveEntry(entry)
	if err != nil {
		s.e.NotifyError(fmt.Errorf("failed to save entry: %w", err))
		return
	}

	err = s.e.Build(false)
	if err != nil {
		s.e.NotifyError(fmt.Errorf("failed to build: %w", err))
	}
}

// TODO: move this to eagle package. See how to retrieve the HTML there. After all,
// the static directory is generated in eagle and not here.
func (s *Server) getWebmentionTargets(entry *eagle.Entry) ([]string, error) {
	s.staticFsLock.RLock()
	// NOTE: instead of using .readHTML here, it would be very interesting to extract
	// it directly from the markdown. However, there are things on the markdown, such as
	// shortcodes that end up generating more HTML with possibly more links.
	html, err := s.staticFs.readHTML(entry.ID)
	s.staticFsLock.RUnlock()
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(html)

	targets, err := webmention.DiscoverLinksFromReader(r, entry.Permalink, ".h-entry .e-content a")
	if err != nil {
		return nil, err
	}

	if entry.Metadata.ReplyTo != nil && entry.Metadata.ReplyTo.URL != "" {
		targets = append(targets, entry.Metadata.ReplyTo.URL)
	}

	return targets, nil
}

func (s *Server) goWebmentions(entry *eagle.Entry) {
	var err error

	defer func() {
		if err != nil {
			s.e.NotifyError(fmt.Errorf("webmentions: %w", err))
		}
	}()

	targets, err := s.getWebmentionTargets(entry)
	if err != nil {
		err = fmt.Errorf("could not fetch webmention targets for %s: %w", entry.ID, err)
		return
	}

	s.Infow("webmentions: found targets", "entry", entry.ID, "permalink", entry.Permalink, "targets", targets)
	err = s.e.SendWebmention(entry.Permalink, targets...)
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
