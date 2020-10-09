package services

import (
	"path"
	"sync"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/hacdias/eagle/config"
)

type Services struct {
	cfg         *config.Config
	Git         *GitPlacebo
	Hugo        *Hugo
	Media       *Media
	Notify      *Notify
	Webmentions *Webmentions
	XRay        *XRay
	Twitter     *twitter.Client
}

func NewServices(c *config.Config) (*Services, error) {
	mutex := &sync.Mutex{}

	notify, err := NewNotify(&c.Telegram)
	if err != nil {
		return nil, err
	}

	// git := &Git{
	// 	Mutex:     mutex,
	// 	Directory: c.Hugo.Source,
	// }

	git := &GitPlacebo{}

	hugo := &Hugo{
		Mutex: mutex,
		Hugo:  c.Hugo,
	}

	media := &Media{c.BunnyCDN}

	config := oauth1.NewConfig(c.Twitter.Key, c.Twitter.Secret)
	token := oauth1.NewToken(c.Twitter.Token, c.Twitter.TokenSecret)

	bird := twitter.NewClient(config.Client(oauth1.NoContext, token))

	return &Services{
		cfg:    c,
		Git:    git,
		Hugo:   hugo,
		Media:  media,
		Notify: notify,
		Webmentions: &Webmentions{
			Mutex:     mutex,
			Domain:    c.Domain,
			Telegraph: c.Telegraph,
			Git:       git,
			Hugo:      hugo,
			Media:     media,
		},
		XRay: &XRay{
			Mutex:       mutex,
			XRay:        c.XRay,
			Twitter:     c.Twitter,
			StoragePath: path.Join(c.Hugo.Source, "data", "xray"),
		},
		Twitter: bird,
	}, nil
}
