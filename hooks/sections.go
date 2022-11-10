package hooks

import (
	"strings"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
)

type SectionDeducer map[mf2.Type][]string

func (s SectionDeducer) DeduceSections(e *eagle.Entry) {
	if len(s) == 0 || len(e.Sections) != 0 {
		return
	}

	mm := e.Helper()
	postType := mm.PostType()

	// Only add the sections to entries under the /year/month/date.
	// This avoids adding sections to top-level pages that shouldn't
	// have these sections.
	if strings.HasPrefix(e.ID, "/20") {
		if sections, ok := s[postType]; ok {
			e.Sections = append(e.Sections, sections...)
		}
	}
}

func (s SectionDeducer) EntryHook(e *eagle.Entry, isNew bool) error {
	if isNew && e.Listing == nil {
		s.DeduceSections(e)
	}

	return nil
}
