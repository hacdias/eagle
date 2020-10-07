package services

import (
	"sync"

	"github.com/hacdias/eagle/config"
)

type Services struct {
	Git         *Git
	Hugo        *Hugo
	Media       *Media
	Notify      *Notify
	Webmentions *Webmentions
	XRay        *XRay
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
	}
}
