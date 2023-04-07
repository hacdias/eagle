package webmentions

import (
	"net/http"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/log"
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
}

func NewWebmentions(fs *fs.FS, notifier eagle.Notifier, renderer *renderer.Renderer) *Webmentions {
	return &Webmentions{
		log: log.S().Named("webmentions"),
		client: webmention.New(&http.Client{
			Timeout: time.Minute,
		}),
		fs:       fs,
		notifier: notifier,
		renderer: renderer,
	}
}

func (ws *Webmentions) EntryHook(old, new *eagle.Entry) error {
	return ws.SendWebmentions(old, new)
}
