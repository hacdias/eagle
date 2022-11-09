package hooks

import (
	"fmt"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/pkg/xray"
)

type ContextXRay struct {
	XRay  *xray.XRay
	Eagle *eagle.Eagle // WIP: remove this once possible.
}

func (c *ContextXRay) EntryHook(e *entry.Entry, isNew bool) error {
	return c.EnsureXRay(e, false)
}

func (c *ContextXRay) EnsureXRay(e *entry.Entry, replace bool) error {
	mm := e.Helper()
	typ := mm.PostType()

	switch typ {
	case mf2.TypeLike,
		mf2.TypeRepost,
		mf2.TypeReply,
		mf2.TypeRsvp:
		// Keep going
	default:
		return nil
	}

	property := mm.TypeProperty()
	if typ == mf2.TypeRsvp {
		property = "in-reply-to"
	}

	urlStr := mm.String(property)
	if urlStr == "" {
		return fmt.Errorf("expected context url to be non-empty for %s", e.ID)
	}

	sidecar, err := c.Eagle.GetSidecar(e)
	if err != nil {
		return fmt.Errorf("could not fetch sidecar for %s: %w", e.ID, err)
	}

	if sidecar.Context != nil && !replace {
		return nil
	}

	parsed, _, err := c.XRay.Fetch(urlStr)
	if err != nil {
		return fmt.Errorf("could not fetch context xray for %s: %w", e.ID, err)
	}

	if parsed.Author.Photo != "" {
		parsed.Author.Photo = c.Eagle.SafeUploadFromURL("wm", parsed.Author.Photo, true)
	}

	return c.Eagle.UpdateSidecar(e, func(data *eagle.Sidecar) (*eagle.Sidecar, error) {
		data.Context = parsed
		return data, nil
	})
}
