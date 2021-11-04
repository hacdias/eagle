package mf2

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
func DiscoverType(data map[string]interface{}) (Type, string) {
	properties := typed.New(data)
	typ := getType(properties)

	switch typ {
	case "event", "recipe", "review":
		return Type(typ), ""
	}

	// This ensures that we can either send a MF2 post, a JF2 post or even an XRay post.
	if p, ok := properties.MapIf("properties"); ok {
		properties = typed.New(p)
	}

	for prop, typ := range propertyToType {
		if _, ok := properties[prop]; ok {
			return typ, prop
		}
	}

	name, ok := properties.StringIf("name")
	if !ok || name == "" {
		return TypeNote, ""
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
		return TypeArticle, ""
	}

	return TypeNote, ""
}

// IsType checks if the given type is a valid Microformats type.
func IsType(typ Type) bool {
	t := Type(typ)
	switch t {
	case TypeRsvp, TypeRepost, TypeLike, TypeReply, TypeBookmark,
		TypeFollow, TypeRead, TypeWatch, TypeListen, TypeCheckin, TypeVideo,
		TypeAudio, TypePhoto, TypeEvent, TypeRecipe, TypeReview, TypeNote, TypeArticle:
		return true
	default:
		return false
	}
}

func getType(data typed.Typed) string {
	var typ string

	if t, ok := data.StringIf("type"); ok {
		typ = t
	}

	if ts, ok := data.StringsIf("type"); ok {
		typ = strings.TrimSpace(strings.Join(ts, ""))
	}

	typ = strings.TrimPrefix(typ, "h-")
	return typ
}
