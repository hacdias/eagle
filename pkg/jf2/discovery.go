package jf2

import (
	"regexp"
	"strings"

	"github.com/karlseguin/typed"
)

// Type represents a post type.
type Type string

const (
	TypeRsvp     Type = "rsvp"
	TypeRepost   Type = "repost"
	TypeLike     Type = "like"
	TypeReply    Type = "reply"
	TypeBookmark Type = "bookmark"
	TypeFollow   Type = "follow"
	TypeRead     Type = "read"
	TypeWatch    Type = "watch"
	TypeListen   Type = "listen"
	TypeCheckin  Type = "checkin"
	TypeVideo    Type = "video"
	TypeAudio    Type = "audio"
	TypePhoto    Type = "photo"
	TypeEvent    Type = "event"
	TypeRecipe   Type = "recipe"
	TypeReview   Type = "review"
	TypeNote     Type = "note"
	TypeArticle  Type = "article"
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
	"listen-of":   TypeListen,
	"checkin":     TypeCheckin,
	"video":       TypeVideo,
	"audio":       TypeAudio,
	"photo":       TypePhoto,
}

// DiscoverType discovers a post type from its properties according to the algorithm
// described in the "Post Type Discovery" specification.
// 	- https://www.w3.org/TR/post-type-discovery/
// 	- https://indieweb.org/post-type-discovery
//
// This is a slightly modified version of @aaronpk's code to include reads and watches.
// Original code: https://github.com/aaronpk/XRay/blob/master/lib/XRay/PostType.php
func DiscoverType(props map[string]interface{}) Type {
	properties := typed.New(props)

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

	name, exists := properties.StringIf("name")
	if !exists || name == "" {
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
	if strings.Index(content, name) != 0 {
		return TypeArticle
	}

	return TypeNote
}
