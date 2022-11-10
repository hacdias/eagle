package hooks

import "github.com/hacdias/eagle/v4/eagle"

type IgnoreListing struct {
	Hook eagle.EntryHook
}

func (i *IgnoreListing) EntryHook(e *eagle.Entry, isNew bool) error {
	if e.Listing != nil {
		return nil
	}

	return i.Hook.EntryHook(e, isNew)
}
