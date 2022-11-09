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

// postSaveHooks runs post saving hooks. These hooks are non-blocking and the error
// of one does not prevent the execution of others. The implementer should be careful
// to make sure they save the changes.
func (s *Server) postSaveHooks(e *entry.Entry, isNew bool, syndicators []string) {
	err := s.syndicate(e, syndicators)
	if err != nil {
		s.Error(err)
	}

	for _, hook := range s.PostSaveHooks {
		err := hook.EntryHook(e, isNew)
		if err != nil {
			s.Error(err)
		}
	}

	s.Eagle.PostSaveEntry(e)
}

func (s *Server) syndicate(e *entry.Entry, syndicators []string) error {
	syndications, err := s.syndicator.Syndicate(e, syndicators)
	if err != nil {
		return err
	}

	if len(syndications) == 0 {
		return nil
	}

	_, err = s.Eagle.TransformEntry(e.ID, func(e *entry.Entry) (*entry.Entry, error) {
		mm := e.Helper()
		syndications := append(mm.Strings("syndication"), syndications...)
		e.Properties["syndication"] = syndications
		return e, nil
	})
	return err
}
