package services

import (
	"path"

	"github.com/hacdias/eagle/config"
)

type Services struct {
	PublicDirChanges chan string
	cfg              *config.Config
	Store            StorageService
	Hugo             *Hugo
	Media            *Media
	Notify           *Notify
	Webmentions      *Webmentions
	XRay             *XRay
	Syndicator       Syndicator
	MeiliSearch      *MeiliSearch
	ActivityPub      *ActivityPub
}

func NewServices(c *config.Config) (*Services, error) {
	notify, err := NewNotify(&c.Telegram, c.S().Named("telegram"))
	if err != nil {
		return nil, err
	}

	var store StorageService
	if c.Development {
		store = &PlaceboStorage{}
	} else {
		store = &GitStorage{
			Directory: c.Hugo.Source,
		}
	}

	dirChanges := make(chan string)

	hugo := &Hugo{
		SugaredLogger: c.S().Named("hugo"),
		Hugo:          c.Hugo,
		Domain:        c.Domain,
		DirChanges:    dirChanges,
	}

	media := &Media{c.BunnyCDN}

	webmentions := &Webmentions{
		SugaredLogger: c.S().Named("webmentions"),
		Domain:        c.Domain,
		Telegraph:     c.Telegraph,
		Hugo:          hugo,
		Media:         media,
	}

	activitypub, err := NewActivityPub(c)
	if err != nil {
		return nil, err
	}
	activitypub.Webmentions = webmentions

	syndicator := Syndicator{}

	if c.Twitter.User != "" {
		twitter := NewTwitter(&c.Twitter)
		syndicator["https://twitter.com/"+c.Twitter.User] = twitter
		hugo.Twitter = twitter
	}

	services := &Services{
		PublicDirChanges: dirChanges,
		cfg:              c,
		Store:            store,
		Hugo:             hugo,
		Media:            media,
		Notify:           notify,
		Webmentions:      webmentions,
		XRay: &XRay{
			SugaredLogger: c.S().Named("xray"),
			XRay:          c.XRay,
			Twitter:       c.Twitter,
			StoragePath:   path.Join(c.Hugo.Source, "data", "xray"),
		},
		Syndicator:  syndicator,
		ActivityPub: activitypub,
	}

	if c.MeiliSearch != nil {
		services.MeiliSearch, err = NewMeiliSearch(c.MeiliSearch)
		if err != nil {
			return nil, err
		}
	}

	return services, nil
}
