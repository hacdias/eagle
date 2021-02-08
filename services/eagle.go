package services

import "github.com/hacdias/eagle/config"

type Eagle struct {
	PublicDirCh chan string

	StorageService
	*EntryManager
	*Media
	*Notifications
	*Hugo
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	media := &Media{conf.BunnyCDN}

	publicDirCh := make(chan string)

	notifications, err := NewNotifications(&conf.Telegram, conf.S().Named("telegram"))
	if err != nil {
		return nil, err
	}

	entryManager := &EntryManager{
		domain: conf.Domain,
		source: conf.Hugo.Source,
	}

	hugo := &Hugo{
		SugaredLogger: conf.S().Named("hugo"),
		Hugo:          conf.Hugo,
		PublicDirCh:   publicDirCh,
	}

	var store StorageService
	if conf.Development {
		store = &PlaceboStorage{}
	} else {
		store = &GitStorage{
			Directory: conf.Hugo.Source,
		}
	}

	return &Eagle{
		PublicDirCh:    publicDirCh,
		EntryManager:   entryManager,
		Media:          media,
		Notifications:  notifications,
		Hugo:           hugo,
		StorageService: store,
	}, nil
}
