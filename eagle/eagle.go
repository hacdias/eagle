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
	log *zap.SugaredLogger

	srcFs     *afero.Afero
	srcGit    *gitRepo
	entriesMu sync.RWMutex

	dstFs            *afero.Afero
	buildMu          sync.Mutex
	currentPublicDir string

	webmentionsClient *webmention.Client
	webmentionsMu     sync.Mutex

	Notifications
	Config      *config.Config
	PublicDirCh chan string

	// Optional services
	media  *Media
	search SearchIndex

	Miniflux *Miniflux
	Twitter  *Twitter
}

func NewEagle(conf *config.Config) (eagle *Eagle, err error) {
	eagle = &Eagle{
		log:    logging.S().Named("eagle"),
		srcFs:  makeAfero(conf.SourceDirectory),
		srcGit: &gitRepo{conf.SourceDirectory},
		dstFs:  makeAfero(conf.PublicDirectory),
		webmentionsClient: webmention.New(&http.Client{
			Timeout: time.Minute,
		}),

		Config:      conf,
		PublicDirCh: make(chan string),
	}

	if conf.BunnyCDN != nil {
		eagle.media = &Media{conf.BunnyCDN}
	}

	if conf.Telegram != nil {
		notifications, err := newTgNotifications(conf.Telegram)
		if err != nil {
			return nil, err
		}
		eagle.Notifications = notifications
	} else {
		eagle.Notifications = newLogNotifications()
	}

	if conf.MeiliSearch != nil {
		search, indexOk, err := NewMeiliSearch(conf.MeiliSearch)
		if err != nil {
			return nil, err
		}
		eagle.search = search

		if !indexOk {
			defer func() {
				logging.S().Info("building index for the first time")
				err = eagle.RebuildIndex()
			}()
		}
	}

	if conf.Twitter != nil {
		eagle.Twitter = NewTwitter(conf.Twitter)
	}

	if conf.Miniflux != nil {
		eagle.Miniflux = &Miniflux{Miniflux: conf.Miniflux}
	}

	return eagle, err
}

func makeAfero(path string) *afero.Afero {
	return &afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), path),
	}
}
