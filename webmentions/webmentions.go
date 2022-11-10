package webmentions

import (
	"net/http"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/log"
	"github.com/hacdias/eagle/media"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/pkg/xray"
	"github.com/hacdias/eagle/renderer"
	"go.uber.org/zap"
	"willnorris.com/go/webmention"
)

type Webmentions struct {
	log      *zap.SugaredLogger
	client   *webmention.Client
	fs       *fs.FS
	notifier eagle.Notifier
	renderer *renderer.Renderer
	media    *media.Media
}

func NewWebmentions(fs *fs.FS, notifier eagle.Notifier, renderer *renderer.Renderer, media *media.Media) *Webmentions {
	return &Webmentions{
		log: log.S().Named("webmentions"),
		client: webmention.New(&http.Client{
			Timeout: time.Minute,
		}),
		fs:       fs,
		notifier: notifier,
		renderer: renderer,
		media:    media,
	}
}

func (ws *Webmentions) EntryHook(e *eagle.Entry, isNew bool) error {
	if e.Listing != nil {
		return nil
	}

	return ws.SendWebmentions(e)
}

func IsInteraction(post *xray.Post) bool {
	return post.Type == mf2.TypeLike ||
		post.Type == mf2.TypeRepost ||
		post.Type == mf2.TypeBookmark ||
		post.Type == mf2.TypeRsvp
}
