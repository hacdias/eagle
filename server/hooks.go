package server

import "github.com/hacdias/eagle/v4/eagle"

// preSaveEntry runs pre saving hooks. These hooks are blocking and they stop
// at the first error. All changes made to the entry in these hooks is saved
// by the caller.
func (s *Server) preSaveEntry(e *eagle.Entry, isNew bool) error {
	for _, hook := range s.preSaveHooks {
		err := hook.EntryHook(e, isNew)
		if err != nil {
			return err
		}
	}

	return nil
}

// postSaveEntry runs post saving hooks. These hooks are non-blocking and the error
// of one does not prevent the execution of others. The implementer should be careful
// to make sure they save the changes.
func (s *Server) postSaveEntry(e *eagle.Entry, isNew bool, syndicators []string) {
	err := s.syndicate(e, syndicators)
	if err != nil {
		s.n.Error(err)
	}

	for _, hook := range s.postSaveHooks {
		err := hook.EntryHook(e, isNew)
		if err != nil {
			s.n.Error(err)
		}
	}

	s.cache.Delete(e)
}

func (s *Server) syndicate(e *eagle.Entry, syndicators []string) error {
	syndications, err := s.syndicator.Syndicate(e, syndicators)
	if err != nil {
		return err
	}

	if len(syndications) == 0 {
		return nil
	}

	_, err = s.fs.TransformEntry(e.ID, func(e *eagle.Entry) (*eagle.Entry, error) {
		mm := e.Helper()
		syndications := append(mm.Strings("syndication"), syndications...)
		e.Properties["syndication"] = syndications
		return e, nil
	})
	return err
}
