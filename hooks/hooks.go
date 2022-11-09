package hooks

import "github.com/hacdias/eagle/v4/entry"

// wip: find better place to put this (with common types?)
type EntryHook interface {
	EntryHook(e *entry.Entry, isNew bool) error
}
