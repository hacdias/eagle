package eagle

import (
	"net/http"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
	"github.com/spf13/afero"
	"willnorris.com/go/webmention"
)

type Eagle struct {
	// srcFs *Storage
	// dstFs *Storage

	PublicDirCh chan string
	Twitter     *Twitter
	Miniflux    *Miniflux

	*Notifications
	*Webmentions
	*EntryManager
	*Hugo
	*Crawler

	*Storage
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	publicDirCh := make(chan string)

	notifications, err := NewNotifications(&conf.Telegram)
	if err != nil {
		return nil, err
	}

	srcFs := NewStorage(conf.Hugo.Source, &gitRepo{
		dir: conf.Hugo.Source,
	})

	webmentions := &Webmentions{
		log:    logging.S().Named("webmentions"),
		media:  &Media{conf.BunnyCDN},
		notify: notifications,
		store:  srcFs.Sub("content"),
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

		PublicDirCh: publicDirCh,
		EntryManager: &EntryManager{
			baseURL: conf.BaseURL,
			store:   srcFs.Sub("content"),
			search:  search,
		},
		Notifications: notifications,
		Hugo: &Hugo{
			conf: conf.Hugo,
			dstFs: &afero.Afero{
				Fs: afero.NewBasePathFs(afero.NewOsFs(), conf.Hugo.Destination),
			},
			publicDirCh: publicDirCh,
		},
		Storage: srcFs,
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
