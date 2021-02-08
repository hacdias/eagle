package services

import "github.com/hacdias/eagle/config"

type Eagle struct {
	*EntryManager
}

func NewEagle(conf *config.Config) (*Eagle, error) {
	entryManager := &EntryManager{
		domain: conf.Domain,
		source: conf.Hugo.Source,
	}

	eagle := &Eagle{
		EntryManager: entryManager,
	}

	return eagle, nil
}
