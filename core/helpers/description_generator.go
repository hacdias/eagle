package helpers

import (
	"net/url"

	"github.com/hacdias/eagle/core"
)

func GenerateDescription(e *core.Entry, replaceDescription bool) {
	if e.Description != "" && !replaceDescription {
		return
	}

	if e.Reply != "" {
		e.Description = "Replied to a post on " + domain(e.Reply)
	}
}

func domain(text string) string {
	u, err := url.Parse(text)
	if err != nil {
		return text
	}

	return u.Host
}
