package services

import (
	"path"

	"github.com/hacdias/eagle/config"
)

type Services struct {
	cfg         *config.Config
	Store       StorageService
	Hugo        *Hugo
	Media       *Media
	Notify      *Notify
	Webmentions *Webmentions
	XRay        *XRay
	Syndicator  Syndicator
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

	hugo := &Hugo{
		Hugo: c.Hugo,
	}

	media := &Media{c.BunnyCDN}

	syndicator := Syndicator{}

	if c.Twitter.User != "" {
		syndicator["https://twitter.com/"+c.Twitter.User] = NewTwitter(&c.Twitter)
	}

	return &Services{
		cfg:    c,
		Store:  store,
		Hugo:   hugo,
		Media:  media,
		Notify: notify,
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
	}, nil
}
