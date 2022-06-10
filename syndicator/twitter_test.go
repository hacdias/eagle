package syndicator

import (
	"testing"

	"github.com/hacdias/eagle/v4/config"
	"github.com/hacdias/eagle/v4/entry"
)

func newTestTwitter() Syndicator {
	return NewTwitter(&config.Twitter{
		User:        "test",
		Key:         "key",
		Secret:      "secret",
		Token:       "token",
		TokenSecret: "token-secret",
	})
}

var isNotByContext = []*entry.Entry{
	{
		Frontmatter: entry.Frontmatter{
			Properties: map[string]interface{}{
				"syndication": "https://twitter.com/status/some-status",
				"like-of":     "https://twitter.com/status/some-status",
			},
		},
	},
	{
		Frontmatter: entry.Frontmatter{
			Properties: map[string]interface{}{
				"invalid-property": "https://twitter.com/status/some-status",
			},
		},
	},
	{
		Frontmatter: entry.Frontmatter{
			Properties: map[string]interface{}{
				"bookmark-of": "https://twitter.com/status/some-status",
			},
		},
	},
}

func TestTwitterIsNotByContext(t *testing.T) {
	twitter := newTestTwitter()

	for _, ee := range isNotByContext {
		if twitter.IsByContext(ee) {
			t.Error("twitter.IsByContext should be false")
		}
	}
}

var isByContext = []*entry.Entry{
	{
		Frontmatter: entry.Frontmatter{
			Properties: map[string]interface{}{
				"like-of": "https://twitter.com/status/some-status",
			},
		},
	},
	{
		Frontmatter: entry.Frontmatter{
			Properties: map[string]interface{}{
				"in-reply-to": []string{"https://twitter.com/status/some-status"},
			},
		},
	},
	{
		Frontmatter: entry.Frontmatter{
			Properties: map[string]interface{}{
				"repost-of": "https://twitter.com/status/some-status",
			},
		},
	},
}

func TestTwitterIsByContext(t *testing.T) {
	twitter := newTestTwitter()

	for _, ee := range isByContext {
		if !twitter.IsByContext(ee) {
			t.Error("twitter.IsByContext should be true")
		}
	}
}
