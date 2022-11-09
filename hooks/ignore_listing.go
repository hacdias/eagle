package hooks

import "github.com/hacdias/eagle/v4/entry"

type IgnoreListing struct {
	Hook EntryHook
}

func (i *IgnoreListing) EntryHook(e *entry.Entry, isNew bool) error {
	if e.Listing != nil {
		return nil
	}

	return i.Hook.EntryHook(e, isNew)
}
