package server

import "github.com/hacdias/eagle/v4/entry"

// PreSaveHook is run before saving an entry.
type PreSaveHook interface {
	PreSaveHook(e *entry.Entry, isNew bool) error
}

// PostSaveHook is run after saving an entry.
type PostSaveHook interface {
	PostSaveHook(e *entry.Entry, isNew bool) error
}

func (s *Server) preSaveEntry(ee *entry.Entry, isNew bool) error {
	for _, hook := range s.PreSaveHooks {
		err := hook.PreSaveHook(ee, isNew)
		if err != nil {
			return err
		}
	}

	if isNew {
		return s.Eagle.PreCreateEntry(ee)
	}

	return nil
}

func (s *Server) postSaveHooks(ee *entry.Entry, isNew bool, syndicators []string) {
	for _, hook := range s.PostSaveHooks {
		err := hook.PostSaveHook(ee, isNew)
		if err != nil {
			s.Error(err)
		}
	}

	s.Eagle.PostSaveEntry(ee, syndicators)
}
