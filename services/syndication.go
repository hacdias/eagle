package services

import (
	"fmt"

	"github.com/hacdias/eagle/middleware/micropub"
	"github.com/hashicorp/go-multierror"
)

type SyndicationService interface {
	Syndicate(entry *HugoEntry, typ micropub.Type, related string) (string, error)
	IsRelated(url string) bool
}

type Syndication struct {
	Type    micropub.Type
	Related []string
	Targets []string
}

type Syndicator map[string]SyndicationService

func (s Syndicator) Syndicate(entry *HugoEntry, synd *Syndication) ([]string, error) {
	var errors *multierror.Error
	var syndications []string

	for _, target := range synd.Targets {
		if service, ok := s[target]; ok {
			url, err := service.Syndicate(entry, synd.Type, "")
			if err != nil {
				errors = multierror.Append(errors, err)
			} else {
				syndications = append(syndications, url)
			}
		} else {
			errors = multierror.Append(errors, fmt.Errorf("unknown syndication service: %s", target))
		}
	}

	for _, url := range synd.Related {
		for _, service := range s {
			if service.IsRelated(url) {
				url, err := service.Syndicate(entry, synd.Type, url)
				if err != nil {
					errors = multierror.Append(errors, err)
				} else {
					syndications = append(syndications, url)
				}
			}
		}
	}

	return syndications, errors.ErrorOrNil()
}
