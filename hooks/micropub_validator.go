package hooks

import (
	"strings"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/samber/lo"
)

type MicropubValidator struct {
	Sections map[mf2.Type][]string
	Unlisted []mf2.Type
}

func NewMicropubValidator(m eagle.Micropub) *MicropubValidator {
	return &MicropubValidator{
		Sections: m.Sections,
		Unlisted: m.Unlisted,
	}
}

func (m MicropubValidator) DeduceSections(e *eagle.Entry) {
	if len(m.Sections) == 0 || len(e.Sections) != 0 {
		return
	}

	mm := e.Helper()
	postType := mm.PostType()

	// Only add the sections to entries under the /year/month/date.
	// This avoids adding sections to top-level pages that shouldn't
	// have these sections.
	if strings.HasPrefix(e.ID, "/20") {
		if sections, ok := m.Sections[postType]; ok {
			e.Sections = append(e.Sections, sections...)
		}
	}
}

func (m MicropubValidator) MarkUnlisted(e *eagle.Entry) {
	if len(m.Unlisted) == 0 {
		return
	}

	mm := e.Helper()
	postType := mm.PostType()

	if lo.Contains(m.Unlisted, postType) {
		e.Unlisted = true
	}
}

func (m *MicropubValidator) EntryHook(old, new *eagle.Entry) error {
	if old == nil && new.Listing == nil {
		m.DeduceSections(new)
		m.MarkUnlisted(new)
	}

	return nil
}
