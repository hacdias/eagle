package services

import (
	"path"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
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
	Twitter     *twitter.Client
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

	config := oauth1.NewConfig(c.Twitter.Key, c.Twitter.Secret)
	token := oauth1.NewToken(c.Twitter.Token, c.Twitter.TokenSecret)

	bird := twitter.NewClient(config.Client(oauth1.NoContext, token))

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
		Twitter: bird,
	}, nil
}
