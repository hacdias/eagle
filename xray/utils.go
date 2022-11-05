package xray

import (
	"regexp"
	"strings"

	"github.com/hacdias/eagle/v4/entry/mf2"
)

type xrayResponse struct {
	Data map[string]interface{} `json:"data"`
	Code int                    `json:"code"`
}

var propertyToType = map[string]mf2.Type{
	"like-of":     mf2.TypeLike,
	"repost-of":   mf2.TypeRepost,
	"in-reply-to": mf2.TypeReply,
	"bookmark-of": mf2.TypeBookmark,
	"rsvp":        mf2.TypeRsvp,
}

var (
	spaceCollapser = regexp.MustCompile(`\s+`)
)

func cleanContent(data string) string {
	data = strings.TrimSpace(data)
	data = spaceCollapser.ReplaceAllString(data, " ") // Collapse whitespaces
	return data
}
