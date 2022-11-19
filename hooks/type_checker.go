package hooks

import (
	"errors"
	"fmt"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/samber/lo"
)

var ErrTypeNotAllowed = errors.New("type not allowed")

type TypeChecker []mf2.Type

func (a TypeChecker) EntryHook(old, new *eagle.Entry) error {
	if old == nil && new.Listing == nil {
		postType := new.Helper().PostType()
		if !lo.Contains(a, postType) {
			return fmt.Errorf("%w: %s", ErrTypeNotAllowed, postType)
		}
	}

	return nil
}
