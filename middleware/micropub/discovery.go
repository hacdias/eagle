package micropub

import (
	"regexp"
	"strings"

	"github.com/karlseguin/typed"
)

// Type represents a post type.
type Type string

const (
	TypeRsvp     Type = "rsvp"
	TypeRepost        = "repost"
	TypeLike          = "like"
	TypeReply         = "reply"
	TypeBookmark      = "bookmark"
	TypeFollow        = "follow"
	TypeRead          = "read"
	TypeWatch         = "watch"
	TypeCheckin       = "checkin"
	TypeVideo         = "video"
	TypeAudio         = "audio"
	TypePhoto         = "photo"
	TypeEvent         = "event"
	TypeRecipe        = "recipe"
	TypeReview        = "review"
	TypeNote          = "note"
	TypeArticle       = "article"
)

var propertyToType = map[string]Type{
	"rsvp":        TypeRsvp,
	"repost-of":   TypeRepost,
	"like-of":     TypeLike,
	"in-reply-to": TypeReply,
	"bookmark-of": TypeBookmark,
	"follow-of":   TypeFollow,
	"read-of":     TypeRead,
	"watch-of":    TypeWatch,
	"checkin":     TypeCheckin,
	"video":       TypeVideo,
	"audio":       TypeAudio,
	"photo":       TypePhoto,
}

// DiscoverType discovers a post type from its properties according to the algorithm
// published on W3C: https://www.w3.org/TR/post-type-discovery/
//
// Code highly based on https://github.com/aaronpk/XRay/blob/5b2b4f31425ffe9c68833a26903fd1716b75717a/lib/XRay/PostType.php
func DiscoverType(properties typed.Typed) Type {
	if typ, ok := properties.StringIf("type"); ok {
		switch typ {
		case "event", "recipe", "review":
			return Type(typ)
		}
	}

	for key, val := range propertyToType {
		if _, ok := properties[key]; ok {
			return val
		}
	}

	nameSlice, exists := properties.StringsIf("name")
	if !exists {
		return TypeNote
	}

	name := strings.TrimSpace(strings.Join(nameSlice, ""))
	if name == "" {
		return TypeNote
	}

	content := ""
	if val, ok := properties.MapIf("content"); ok {
		content = val["text"].(string)
	} else if val, ok := properties.StringIf("summary"); ok {
		content = val
	}

	// Collapse all sequences of internal whitespace to a single space (0x20) character each
	var re = regexp.MustCompile(`/\s+/`)
	name = re.ReplaceAllString(name, " ")
	content = re.ReplaceAllString(content, " ")

	// If this processed "name" property value is NOT a prefix of the
	// processed "content" property, then it is an article post.
	if strings.Index(content, name) == -1 {
		return TypeArticle
	}

	return TypeNote
}
