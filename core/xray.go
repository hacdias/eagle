package core

import (
	"io"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"go.hacdias.com/indielib/microformats"
	wmf "willnorris.com/go/microformats"
)

type XRay struct {
	Name        string            `json:"name,omitempty"`
	Content     string            `json:"content,omitempty"`
	Author      string            `json:"author,omitempty"`
	AuthorPhoto string            `json:"authorPhoto,omitempty"`
	AuthorURL   string            `json:"authorUrl,omitempty"`
	Date        time.Time         `json:"date,omitempty"`
	URL         string            `json:"url,omitempty"`
	Type        microformats.Type `json:"type,omitempty"`
	Private     bool              `json:"private,omitempty"`
}

var (
	htmlPolicy = bluemonday.StrictPolicy()
	spaces     = regexp.MustCompile(`\s+`)
	breaks     = regexp.MustCompile(`<br\s*/?>`)
)

func sanitizeContent(data string) string {
	data = breaks.ReplaceAllString(data, " ")
	data = htmlPolicy.Sanitize(data)
	data = strings.TrimSpace(data)
	// Collapse white spaces.
	data = spaces.ReplaceAllString(data, " ")
	return data
}

func ParseXRay(r io.Reader, u *url.URL) *XRay {
	data := wmf.Parse(r, u)
	sourceURL := u.String()

	// Find the first h-entry item.
	var entry *wmf.Microformat
	for _, item := range data.Items {
		if slices.Contains(item.Type, "h-entry") {
			entry = item
		}

		if entry != nil {
			break
		}
	}

	if entry == nil {
		return &XRay{URL: sourceURL}
	}

	post := &XRay{
		URL:  sourceURL,
		Name: getFirstString(entry.Properties, "name"),
	}

	// u-url
	if urlStr := getFirstString(entry.Properties, "url"); urlStr != "" {
		post.URL = urlStr
	}

	// content: e-content (map with html/text) or p-summary
	if content, ok := entry.Properties["content"]; ok && len(content) > 0 {
		switch v := content[0].(type) {
		case map[string]any:
			if html, ok := v["html"].(string); ok && html != "" {
				post.Content = sanitizeContent(html)
			} else if text, ok := v["value"].(string); ok && text != "" {
				post.Content = sanitizeContent(text)
			}
		case map[string]string:
			if html := v["html"]; html != "" {
				post.Content = sanitizeContent(html)
			} else if text := v["value"]; text != "" {
				post.Content = sanitizeContent(text)
			}
		case string:
			post.Content = sanitizeContent(v)
		}
	}

	if post.Content == "" {
		if summary := getFirstString(entry.Properties, "summary"); summary != "" {
			post.Content = sanitizeContent(summary)
		}
	}

	// dt-published
	if pub := getFirstString(entry.Properties, "published"); pub != "" {
		if t, err := time.Parse(time.RFC3339, pub); err == nil {
			post.Date = t
		}
	}

	// p-author (can be an embedded h-card or a plain string)
	if authors, ok := entry.Properties["author"]; ok && len(authors) > 0 {
		switch v := authors[0].(type) {
		case *wmf.Microformat:
			post.Author = getFirstString(v.Properties, "name")
			post.AuthorURL = getFirstString(v.Properties, "url")
			post.AuthorPhoto = getFirstString(v.Properties, "photo")
		case string:
			post.Author = v
		}
	}

	// Post type via Post Type Discovery.
	post.Type, _ = microformats.DiscoverType(map[string]any{
		"type":       entry.Type,
		"properties": entry.Properties,
	})

	return post
}

func getFirstString(props map[string][]any, key string) string {
	vals, ok := props[key]
	if !ok || len(vals) == 0 {
		return ""
	}

	switch v := vals[0].(type) {
	case string:
		return strings.TrimSpace(v)
	case *wmf.Microformat:
		return strings.TrimSpace(v.Value)
	}

	return ""
}
