package hooks

import (
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/util"
	"github.com/samber/lo"
)

type TagsSanitizer struct{}

func (t TagsSanitizer) EntryHook(_, e *eagle.Entry) error {
	for i := range e.Tags {
		e.Tags[i] = util.Slugify(e.Tags[i])
	}

	e.Tags = lo.Uniq(e.Tags)
	return nil
}
