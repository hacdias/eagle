package hooks

import (
	"errors"
	"fmt"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/thoas/go-funk"
)

var ErrTypeNotAllowed = errors.New("type not allowed")

type AllowedType []mf2.Type

func (a AllowedType) EntryHook(e *entry.Entry, isNew bool) error {
	if isNew {
		postType := e.Helper().PostType()
		if !funk.Contains(a, postType) {
			return fmt.Errorf("%w: %s", ErrTypeNotAllowed, postType)
		}
	}

	return nil
}