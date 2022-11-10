package hooks

import (
	"fmt"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/media"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/pkg/xray"
)

type ContextXRay struct {
	xray  *xray.XRay
	fs    *fs.FS
	media *media.Media
}

func NewContentXRay(xray *xray.XRay, fs *fs.FS, media *media.Media) *ContextXRay {
	return &ContextXRay{
		xray:  xray,
		fs:    fs,
		media: media,
	}
}

func (c *ContextXRay) EntryHook(e *eagle.Entry, isNew bool) error {
	return c.EnsureXRay(e, false)
}

func (c *ContextXRay) EnsureXRay(e *eagle.Entry, replace bool) error {
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

	sidecar, err := c.fs.GetSidecar(e)
	if err != nil {
		return fmt.Errorf("could not fetch sidecar for %s: %w", e.ID, err)
	}

	if sidecar.Context != nil && !replace {
		return nil
	}

	parsed, _, err := c.xray.Fetch(urlStr)
	if err != nil {
		return fmt.Errorf("could not fetch context xray for %s: %w", e.ID, err)
	}

	if parsed.Author.Photo != "" && c.media != nil {
		parsed.Author.Photo = c.media.SafeUploadFromURL("wm", parsed.Author.Photo, true)
	}

	return c.fs.UpdateSidecar(e, func(data *eagle.Sidecar) (*eagle.Sidecar, error) {
		data.Context = parsed
		return data, nil
	})
}
