package eagle

type EntryHook interface {
	EntryHook(old, new *Entry) error
}
