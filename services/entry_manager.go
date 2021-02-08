package services

import "sync"

type EntryManager struct {
	sync.Mutex
}

func NewEntryManager() (*EntryManager, error) {

	return nil, nil
}
