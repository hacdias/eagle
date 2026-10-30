package xray

import (
	"strings"

	"github.com/araddon/dateparse"
	"github.com/karlseguin/typed"
	"go.hacdias.com/indielib/microformats"
)

func Parse(data map[string]any) *Post {
	raw := typed.New(data)

	if raw.String("type") == "feed" {
		items := raw.Maps("items")
		if len(items) >= 1 {
			for _, item := range items {
				if typed.New(item).String("type") == "entry" {
					return Parse(item)
				}
			}

			return Parse(items[0])
		}
	}

	parsed := &Post{
		URL: raw.StringOr("wm-source", raw.String("url")),
	}

	if date := raw.StringOr("published", raw.String("wm-received")); date != "" {
		t, err := dateparse.ParseStrict(date)
		if err == nil {
			parsed.Date = t
		}
	}

	if name, ok := raw.StringIf("name"); ok {
		parsed.Name = name
	}

	var hasContent bool

	if content, ok := raw.StringIf("content"); ok {
		parsed.Content = SanitizeContent(content)
		hasContent = true
	}

	if !hasContent {
		if contentMap, ok := raw.MapIf("content"); ok && !hasContent {
			typedContentMap := typed.New(contentMap)
			if text, ok := typedContentMap.StringIf("text"); ok {
				parsed.Content = SanitizeContent(text)
				hasContent = true
			} else if html, ok := typedContentMap.StringIf("html"); ok {
				parsed.Content = SanitizeContent(html)
				hasContent = true
			}
		}
	}

	if !hasContent {
		if name, ok := raw.StringIf("name"); ok {
			parsed.Content = SanitizeContent(name)
		}
	}

	if photos, ok := raw.StringsIf("photo"); ok {
		parsed.Content += " " + strings.Join(photos, " ")
		parsed.Content = strings.TrimSpace(parsed.Content)
	}

	if author, ok := raw.MapIf("author"); ok {
		typedAuthor := typed.New(author)
		parsed.Author = typedAuthor.String("name")
		parsed.AuthorPhoto = typedAuthor.String("photo")
		parsed.AuthorURL = typedAuthor.String("url")
	}

	if wmProperty, ok := raw.StringIf("wm-property"); ok {
		parsed.Type = microformats.PropertyToType(wmProperty)
	}

	if wmPrivate, ok := raw.BoolIf("wm-private"); ok {
		parsed.Private = wmPrivate
	} else {
		parsed.Private = raw.String("wm-private") == "true"
	}

	return parsed
}
