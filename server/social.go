package server

import (
	"bytes"
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hacdias/eagle/eagle"
)

func (s *Server) goSyndicate(entry *eagle.Entry) {
	if s.e.Twitter == nil {
		return
	}

	url, err := s.e.Twitter.Syndicate(entry)
	if err != nil {
		s.Errorf("failed to syndicate: %w", err)
		s.e.NotifyError(err)
		return
	}

	entry.Metadata.Syndication = append(entry.Metadata.Syndication, url)
	err = s.e.SaveEntry(entry)
	if err != nil {
		s.Errorf("failed to save entry: %w", err)
		s.e.NotifyError(err)
		return
	}

	err = s.e.Build(false)
	if err != nil {
		s.Errorf("failed to build: %w", err)
		s.e.NotifyError(err)
	}
}

func (s *Server) goWebmentions(entry *eagle.Entry) {
	var err error
	defer func() {
		if err != nil {
			s.e.NotifyError(err)
			s.Warnf("webmentions: %w", err)
		}
	}()

	s.staticFsLock.RLock()
	html, err := s.staticFs.readHTML(entry.ID)
	s.staticFsLock.RUnlock()
	if err != nil {
		s.Errorf("could not fetch HTML for %s: %w", entry.ID, err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return
	}

	targets := []string{}

	if entry.Metadata.ReplyTo != nil && entry.Metadata.ReplyTo.URL != "" {
		targets = append(targets, entry.Metadata.ReplyTo.URL)
	}

	doc.Find(".h-entry .e-content a").Each(func(i int, q *goquery.Selection) {
		val, ok := q.Attr("href")
		if !ok {
			return
		}

		u, err := url.Parse(val)
		if err != nil {
			targets = append(targets, val)
			return
		}

		base, err := url.Parse(entry.Permalink)
		if err != nil {
			targets = append(targets, val)
			return
		}

		targets = append(targets, base.ResolveReference(u).String())
	})

	s.Infow("webmentions: found targets", "entry", entry.ID, "permalink", entry.Permalink, "targets", targets)
	err = s.e.SendWebmention(entry.Permalink, targets...)
}

func sanitizeReplyURL(iu string) string {
	if strings.HasPrefix(iu, "https://twitter.com") && strings.Contains(iu, "/status/") {
		u, err := url.Parse(iu)
		if err != nil {
			return iu
		}

		for k := range u.Query() {
			u.Query().Del(k)
		}

		return u.String()
	}

	return iu
}

func sanitizeID(id string) (string, error) {
	if id != "" {
		u, err := url.Parse(id)
		if err != nil {
			return "", err
		}
		id = path.Clean(u.Path)
	}
	return id, nil
}
