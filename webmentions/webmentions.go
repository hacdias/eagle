package webmentions

import (
	"net/http"
	"time"

	"github.com/hacdias/eagle/core"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/log"
	"go.uber.org/zap"
	"willnorris.com/go/webmention"
)

type Webmentions struct {
	log      *zap.SugaredLogger
	client   *webmention.Client
	fs       *fs.FS
	hugo     *core.Hugo
	notifier core.Notifier
}

func NewWebmentions(fs *fs.FS, hugo *core.Hugo, notifier core.Notifier) *Webmentions {
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

func (ws *Webmentions) EntryHook(old, new *core.Entry) error {
	return ws.SendWebmentions(old, new)
}
