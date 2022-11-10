package hooks

import (
	"errors"
	"fmt"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/thoas/go-funk"
)

var ErrTypeNotAllowed = errors.New("type not allowed")

type TypeChecker []mf2.Type

func (a TypeChecker) EntryHook(e *eagle.Entry, isNew bool) error {
	if isNew && e.Listing == nil {
		postType := e.Helper().PostType()
		if !funk.Contains(a, postType) {
			return fmt.Errorf("%w: %s", ErrTypeNotAllowed, postType)
		}
	}

	return nil
}
