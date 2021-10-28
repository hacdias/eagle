package eagle

import (
	"net/http"
	"sync"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	"willnorris.com/go/webmention"
)

type Eagle struct {
	log  *zap.SugaredLogger
	conf *config.Config

	srcFs     *afero.Afero
	srcGit    *gitRepo
	entriesMu sync.RWMutex

	dstFs            *afero.Afero
	buildMu          sync.Mutex
	currentPublicDir string

	webmentionsClient *webmention.Client
	webmentionsMu     sync.Mutex

	media  *Media
	search SearchIndex

	PublicDirCh chan string

	*Notifications
	*Crawler

	// Optional services
	Miniflux *Miniflux
	Twitter  *Twitter
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	notifications, err := NewNotifications(&conf.Telegram)
	if err != nil {
		return nil, err
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
		log:    logging.S().Named("eagle"),
		conf:   conf,
		srcFs:  makeAfero(conf.Hugo.Source),
		srcGit: &gitRepo{conf.Hugo.Source},
		dstFs:  makeAfero(conf.Hugo.Destination),
		webmentionsClient: webmention.New(&http.Client{
			Timeout: time.Minute,
		}),
		media:  &Media{conf.BunnyCDN},
		search: search,

		PublicDirCh:   make(chan string),
		Notifications: notifications,
		Crawler: &Crawler{
			xray:    conf.XRay,
			twitter: conf.Twitter,
		},
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

func makeAfero(path string) *afero.Afero {
	return &afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
	}
}
