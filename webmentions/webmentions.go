package webmentions

import (
	"net/http"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/log"
	"go.uber.org/zap"
	"willnorris.com/go/webmention"
)

type Webmentions struct {
	log      *zap.SugaredLogger
	client   *webmention.Client
	fs       *fs.FS
	hugo     *eagle.Hugo
	notifier eagle.Notifier
}

func NewWebmentions(fs *fs.FS, hugo *eagle.Hugo, notifier eagle.Notifier) *Webmentions {
	return &Webmentions{
		log: log.S().Named("webmentions"),
		client: webmention.New(&http.Client{
			Timeout: time.Minute,
		}),
		fs:       fs,
		hugo:     hugo,
		notifier: notifier,
	}
}

func (ws *Webmentions) EntryHook(old, new *eagle.Entry) error {
	return ws.SendWebmentions(old, new)
}
