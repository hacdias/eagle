package webmentions

import (
	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/pkg/xray"
	"github.com/hacdias/eagle/v4/renderer"
	"willnorris.com/go/webmention"
)

type WebmentionPayload struct {
	Source  string                 `json:"source"`
	Secret  string                 `json:"secret"`
	Deleted bool                   `json:"deleted"`
	Target  string                 `json:"target"`
	Post    map[string]interface{} `json:"post"`
}

type WebmentionsService struct {
	Client   *webmention.Client
	Renderer *renderer.Renderer
	Eagle    *eagle.Eagle // WIP: remove this once possible.
}

func (ws *WebmentionsService) EntryHook(e *entry.Entry, isNew bool) error {
	return ws.SendWebmentions(e)
}

func IsInteraction(post *xray.Post) bool {
	return post.Type == mf2.TypeLike ||
		post.Type == mf2.TypeRepost ||
		post.Type == mf2.TypeBookmark ||
		post.Type == mf2.TypeRsvp
}
