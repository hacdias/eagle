package eagle

import (
	"github.com/hacdias/eagle/config"
)

type Eagle struct {
	PublicDirCh chan string
	ActivityPub *ActivityPub
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
		domain:     conf.Domain,
		hugoSource: conf.Hugo.Source,
		telegraph:  conf.Telegraph,
		media:      &Media{conf.BunnyCDN},
		notify:     notifications,
		store:      store,
	}

	activitypub, err := NewActivityPub(conf, webmentions, notifications)
	if err != nil {
		return nil, err
	}

	var search SearchIndex
	if conf.MeiliSearch != nil {
		search, err = NewMeiliSearch(conf.MeiliSearch)
		if err != nil {
			return nil, err
		}
	}

	eagle := &Eagle{
		PublicDirCh: publicDirCh,
		EntryManager: &EntryManager{
			domain: conf.Domain,
			source: conf.Hugo.Source,
			store:  store,
			search: search,
		},
		Notifications: notifications,
		Hugo: &Hugo{
			conf:        conf.Hugo,
			publicDirCh: publicDirCh,
		},
		StorageService: store,
		Crawler: &Crawler{
			xray:    conf.XRay,
			twitter: conf.Twitter,
		},
		Webmentions: webmentions,
		ActivityPub: activitypub,
	}

	if conf.Twitter.User != "" {
		eagle.Twitter = NewTwitter(&conf.Twitter)
	}

	return eagle, nil
}
