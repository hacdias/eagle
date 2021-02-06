package services

import "github.com/hacdias/eagle/config"

type Services struct {
	*EntryManager
}

func NewServices(c *config.Config) (*Services, error) {
	entryManager, err := NewEntryManager(c)
	if err != nil {
		return nil, err
	}

	services := &Services{
		EntryManager: entryManager,
	}

	return services, nil
}
