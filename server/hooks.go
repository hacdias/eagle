package server

import "github.com/hacdias/eagle/v4/entry"

type EntryHook interface {
	EntryHook(e *entry.Entry, isNew bool) error
}

func (s *Server) preSaveEntry(ee *entry.Entry, isNew bool) error {
	for _, hook := range s.PreSaveHooks {
		err := hook.EntryHook(ee, isNew)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) postSaveHooks(ee *entry.Entry, isNew bool, syndicators []string) {
	for _, hook := range s.PostSaveHooks {
		err := hook.EntryHook(ee, isNew)
		if err != nil {
			s.Error(err)
		}
	}

	s.Eagle.PostSaveEntry(ee, syndicators)
}
