package eagle

import (
	"path/filepath"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
	"github.com/spf13/afero"
)

type Eagle struct {
	PublicDirCh chan string
	Twitter     *Twitter

	*Notifications
	*Webmentions
	*EntryManager
	*Hugo
	*Crawler

	StorageService
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	publicDirCh := make(chan string)

	notifications, err := NewNotifications(&conf.Telegram)
	if err != nil {
		return nil, err
	}

	var store StorageService
	if conf.Development {
		store = &PlaceboStorage{}
	} else {
		store = &GitStorage{
			dir: conf.Hugo.Source,
		}
	}

	webmentions := &Webmentions{
		log:       logging.S().Named("webmentions"),
		domain:    conf.Domain,
		telegraph: conf.Telegraph,
		media:     &Media{conf.BunnyCDN},
		notify:    notifications,
		store:     store,
		fs:        makeAfero(filepath.Join(conf.Hugo.Source, "content")),
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
		PublicDirCh: publicDirCh,
		EntryManager: &EntryManager{
			domain: conf.Domain,
			fs:     makeAfero(filepath.Join(conf.Hugo.Source, "content")),
			store:  store,
			search: search,
		},
		Notifications: notifications,
		Hugo: &Hugo{
			conf:        conf.Hugo,
			dstFs:       makeAfero(conf.Hugo.Destination),
			publicDirCh: publicDirCh,
		},
		StorageService: store,
		Crawler: &Crawler{
			xray:    conf.XRay,
			twitter: conf.Twitter,
		},
		Webmentions: webmentions,
	}

	if conf.Twitter.User != "" {
		eagle.Twitter = NewTwitter(&conf.Twitter)
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
