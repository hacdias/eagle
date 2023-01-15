package hooks

import (
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/util"
)

var typeToDescription = map[mf2.Type]string{
	mf2.TypeReply:    "Replied to ",
	mf2.TypeLike:     "Liked ",
	mf2.TypeRepost:   "Reposted ",
	mf2.TypeBookmark: "Bookmarked ",
	mf2.TypeAte:      "Just ate: ",
	mf2.TypeDrank:    "Just drank: ",
}

type DescriptionGenerator struct {
	fs *fs.FS
}

func NewDescriptionGenerator(fs *fs.FS) *DescriptionGenerator {
	return &DescriptionGenerator{
		fs: fs,
	}
}

func (d *DescriptionGenerator) EntryHook(old, new *eagle.Entry) error {
	if old == nil && new.Listing == nil {
		return d.GenerateDescription(new, false)
	}

	return nil
}

func (d *DescriptionGenerator) GenerateDescription(e *eagle.Entry, replaceDescription bool) error {
	if e.Description != "" && !replaceDescription {
		return nil
	}

	var (
		description string
		err         error
	)

	mm := e.Helper()

	switch mm.PostType() {
	case mf2.TypeReply,
		mf2.TypeLike,
		mf2.TypeRepost,
		mf2.TypeBookmark:
		url := mm.String(mm.TypeProperty())
		urlDomain := util.Domain(url)
		description = typeToDescription[mm.PostType()] + "a post on " + urlDomain
	case mf2.TypeAte, mf2.TypeDrank:
		// Matches Teacup
		food := mm.Sub(mm.TypeProperty())
		description = typeToDescription[mm.PostType()] + food.Name()
	case mf2.TypeCheckin:
		// Matches OwnYourSwarm
		checkin := mm.Sub(mm.TypeProperty())
		description = "At " + checkin.Name()
	case mf2.TypeRsvp:
		description, err = d.generateRsvpDescription(e)
	}

	if err != nil {
		return err
	}

	if description == "" && e.Description != "" {
		return nil
	}

	e.Description = description
	return nil
}

func (d *DescriptionGenerator) generateRsvpDescription(e *eagle.Entry) (string, error) {
	mm := e.Helper()
	rsvp := mm.String(mm.TypeProperty())

	var name string

	sidecar, err := d.fs.GetSidecar(e)
	if err == nil && sidecar.Context != nil && sidecar.Context.Name != "" {
		name = `"` + sidecar.Context.Name + `"`
	} else {
		domain := util.Domain(mm.String("in-reply-to"))
		if domain != "" {
			name = "an event on " + domain
		}
	}

	if name == "" {
		return "", nil
	}

	switch rsvp {
	case "interested":
		return "Interested in " + name, nil
	case "yes":
		return "Going to " + name, nil
	case "no":
		return "Not going to " + name, nil
	case "maybe":
		return "Maybe going to " + name, nil
	}

	return "", nil
}
