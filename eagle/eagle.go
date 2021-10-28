package eagle

import (
	"net/http"
	"sync"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
	"github.com/spf13/afero"
	"willnorris.com/go/webmention"
)

type Eagle struct {
	// srcFs *Storage

	conf *config.Config

	dstFs            *afero.Afero
	buildMu          sync.Mutex
	currentPublicDir string

	PublicDirCh chan string
	Twitter     *Twitter
	Miniflux    *Miniflux

	*Notifications
	*Webmentions
	*EntryManager
	*Crawler

	*Storage
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	publicDirCh := make(chan string)

	notifications, err := NewNotifications(&conf.Telegram)
	if err != nil {
		return nil, err
	}

	srcFs := NewStorage(conf.Hugo.Source)

	webmentions := &Webmentions{
		log:    logging.S().Named("webmentions"),
		media:  &Media{conf.BunnyCDN},
		notify: notifications,
		store:  srcFs,
		client: webmention.New(&http.Client{
			Timeout: time.Minute,
		}),
	}

	var (
		search  SearchIndex
		indexOk bool
	)
	if conf.MeiliSearch != nil {
		search, indexOk, err = NewMeiliSearch(conf.MeiliSearch)
	}
	if err != nil {
		return nil, err
	}

	eagle := &Eagle{
		// srcFs: srcFs,

		conf: conf,
		dstFs: &afero.Afero{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), conf.Hugo.Destination),
		},

		PublicDirCh: publicDirCh,
		EntryManager: &EntryManager{
			baseURL: conf.BaseURL,
			store:   srcFs,
			search:  search,
		},
		Notifications: notifications,
		Storage:       srcFs,
		Crawler: &Crawler{
			xray:    conf.XRay,
			twitter: conf.Twitter,
		},
		Webmentions: webmentions,
	}

	if conf.Twitter != nil {
		eagle.Twitter = NewTwitter(conf.Twitter)
	}

	if conf.Miniflux != nil {
		eagle.Miniflux = &Miniflux{Miniflux: conf.Miniflux}
	}

	if !indexOk {
		logging.S().Info("building index for the first time")
		err = eagle.RebuildIndex()
	}

	return eagle, err
}
