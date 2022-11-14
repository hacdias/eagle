package hooks

import (
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/util"
	"github.com/thoas/go-funk"
)

type TagsSanitizer struct{}

func (t TagsSanitizer) EntryHook(e *eagle.Entry, isNew bool) error {
	if tags, ok := e.Taxonomies["tags"]; ok {
		for i := range tags {
			tags[i] = util.Slugify(tags[i])
		}
		e.Taxonomies["tags"] = funk.UniqString(tags)
	}

	return nil
}
