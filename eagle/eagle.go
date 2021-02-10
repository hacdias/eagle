package eagle

import (
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
)

type Eagle struct {
	PublicDirCh chan string
	ActivityPub *ActivityPub
	Twitter     *Twitter

	*Media
	*Notifications
	*Webmentions
	*EntryManager
	*Hugo
	*Crawler

	StorageService
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	media := &Media{conf.BunnyCDN}

	publicDirCh := make(chan string)

	notifications, err := NewNotifications(&conf.Telegram, logging.S().Named("telegram"))
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
		media:      media,
		notify:     notifications,
		store:      store,
	}

	activitypub, err := NewActivityPub(conf, webmentions)
	if err != nil {
		return nil, err
	}

	if conf.Twitter.User != "" {
		//twitter := NewTwitter(&conf.Twitter)
		//syndicator["twitter"] = twitter
		// hugo.Twitter = twitter
	}

	return &Eagle{
		PublicDirCh: publicDirCh,
		EntryManager: &EntryManager{
			domain: conf.Domain,
			source: conf.Hugo.Source,
			store:  store,
		},
		Media:         media,
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
