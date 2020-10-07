package services

import (
	"sync"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/hacdias/eagle/config"
)

type Services struct {
	Git         *Git
	Hugo        *Hugo
	Media       *Media
	Notify      *Notify
	Webmentions *Webmentions
	XRay        *XRay
	Twitter     *twitter.Client
}

func NewServices(cfg *config.Config) *Services {
	mutex := &sync.Mutex{}

	return &Services{
		Git: &Git{
			Mutex:     mutex,
			Directory: cfg.Hugo.Source,
		},
		Hugo: &Hugo{
			Mutex: mutex,
			Hugo:  cfg.Hugo,
		},
		Media:       &Media{},
		Notify:      &Notify{},
		Webmentions: &Webmentions{},
		XRay: &XRay{
			Mutex: mutex,
		},
		Twitter: createTwitter(cfg),
	}
}

func createTwitter(cfg *config.Config) *twitter.Client {
	config := oauth1.NewConfig(cfg.Twitter.Key, cfg.Twitter.Secret)
	token := oauth1.NewToken(cfg.Twitter.Token, cfg.Twitter.TokenSecret)

	return twitter.NewClient(config.Client(oauth1.NoContext, token))
}
