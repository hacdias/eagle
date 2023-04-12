package server

import (
	"github.com/hacdias/eagle/eagle"
)

// preSaveEntry runs pre saving hooks. These hooks are blocking and they stop
// at the first error. All changes made to the entry in these hooks is saved
// by the caller.
func (s *Server) preSaveEntry(old, new *eagle.Entry) error {
	for _, hook := range s.preSaveHooks {
		err := hook.EntryHook(old, new)
		if err != nil {
			return err
		}
	}

	return nil
}

// postSaveEntry runs post saving hooks. These hooks are non-blocking and the error
// of one does not prevent the execution of others. The implementer should be careful
// to make sure they save the changes.
func (s *Server) postSaveEntry(old, new *eagle.Entry) {
	for _, hook := range s.postSaveHooks {
		err := hook.EntryHook(old, new)
		if err != nil {
			s.n.Error(err)
		}
	}

	if old != nil {
		// s.cache.Delete(old)
	}

	// s.cache.Delete(new)
}
