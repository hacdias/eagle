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
}

func NewServices(c *config.Config) (*Services, error) {
	notify, err := NewNotify(&c.Telegram)
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
		Hugo:       c.Hugo,
		Domain:     c.Domain,
		DirChanges: dirChanges,
	}

	media := &Media{c.BunnyCDN}

	syndicator := Syndicator{}

	if c.Twitter.User != "" {
		syndicator["https://twitter.com/"+c.Twitter.User] = NewTwitter(&c.Twitter)
	}

	services := &Services{
		PublicDirChanges: dirChanges,
		cfg:              c,
		Store:            store,
		Hugo:             hugo,
		Media:            media,
		Notify:           notify,
		Webmentions: &Webmentions{
			Domain:    c.Domain,
			Telegraph: c.Telegraph,
			Hugo:      hugo,
			Media:     media,
		},
		XRay: &XRay{
			XRay:        c.XRay,
			Twitter:     c.Twitter,
			StoragePath: path.Join(c.Hugo.Source, "data", "xray"),
		},
		Syndicator: syndicator,
	}

	if c.MeiliSearch != nil {
		services.MeiliSearch, err = NewMeiliSearch(c.MeiliSearch)
		if err != nil {
			return nil, err
		}
	}

	return services, nil
}
