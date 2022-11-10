package eagle

type EntryHook interface {
	EntryHook(e *Entry, isNew bool) error
}
