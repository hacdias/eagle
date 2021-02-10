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
			Directory: conf.Hugo.Source,
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

	activitypub, err := NewActivityPub(conf, webmentions)
	if err != nil {
		return nil, err
	}

	if conf.Twitter.User != "" {
		// twitter := NewTwitter(&conf.Twitter)
	}

	return &Eagle{
		PublicDirCh: publicDirCh,
		EntryManager: &EntryManager{
			domain: conf.Domain,
			source: conf.Hugo.Source,
			store:  store,
		},
		Notifications: notifications,
		Hugo: &Hugo{
			Hugo:        conf.Hugo,
			publicDirCh: publicDirCh,
		},
		StorageService: store,
		Crawler: &Crawler{
			xray:    conf.XRay,
			twitter: conf.Twitter,
		},
		Webmentions: webmentions,
		ActivityPub: activitypub,
	}, nil
}
