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
	Git         *Git
	Hugo        *Hugo
	Media       *Media
	Notify      *Notify
	Webmentions *Webmentions
	XRay        *XRay
	Twitter     *twitter.Client
}

func NewServices(cfg *config.Config) (*Services, error) {
	mutex := &sync.Mutex{}

	notify, err := NewNotify(&cfg.Telegram)
	if err != nil {
		return nil, err
	}

	return &Services{
		cfg: cfg,
		Git: &Git{
			Mutex:     mutex,
			Directory: cfg.Hugo.Source,
		},
		Hugo: &Hugo{
			Mutex: mutex,
			Hugo:  cfg.Hugo,
		},
		Media:       &Media{cfg.BunnyCDN},
		Notify:      notify,
		Webmentions: &Webmentions{},
		XRay: &XRay{
			XRay:        cfg.XRay,
			Mutex:       mutex,
			Twitter:     cfg.Twitter,
			StoragePath: path.Join(cfg.Hugo.Source, "data", "xray"),
		},
		Twitter: createTwitter(cfg),
	}, nil
}

func createTwitter(cfg *config.Config) *twitter.Client {
	config := oauth1.NewConfig(cfg.Twitter.Key, cfg.Twitter.Secret)
	token := oauth1.NewToken(cfg.Twitter.Token, cfg.Twitter.TokenSecret)

	return twitter.NewClient(config.Client(oauth1.NoContext, token))
}
