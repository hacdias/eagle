package hooks

import (
	"strconv"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/util"
)

var typeToDescription = map[mf2.Type]string{
	mf2.TypeReply:    "Replied to ",
	mf2.TypeLike:     "Liked ",
	mf2.TypeRepost:   "Reposted ",
	mf2.TypeBookmark: "Bookmarked ",
	mf2.TypeAte:      "Just ate: ",
	mf2.TypeDrank:    "Just drank: ",
}

type DescriptionGenerator struct{}

func (d *DescriptionGenerator) EntryHook(e *eagle.Entry, isNew bool) error {
	if isNew {
		return d.GenerateDescription(e, false)
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
	case mf2.TypePhoto:
		description = "A photo"
	case mf2.TypeVideo:
		description = "A video"
	case mf2.TypeAudio:
		description = "An audio"
	case mf2.TypeNote:
		description = "A note"
	case mf2.TypeAte, mf2.TypeDrank:
		// Matches Teacup
		food := mm.Sub(mm.TypeProperty())
		description = typeToDescription[mm.PostType()] + food.Name()
	case mf2.TypeCheckin:
		// Matches OwnYourSwarm
		checkin := mm.Sub(mm.TypeProperty())
		description = "At " + checkin.Name()
	case mf2.TypeItinerary:
		description, err = d.generateItineraryDescription(e)
	case mf2.TypeRsvp:
		description, err = d.generateRsvpDescription(e)
	case mf2.TypeWatch:
		description, err = d.generateWatchDescription(e)
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

func (d *DescriptionGenerator) generateItineraryDescription(ee *eagle.Entry) (string, error) {
	mm := ee.Helper()

	subs := mm.Subs(mm.TypeProperty())
	if len(subs) == 0 {
		return "", nil
	}

	start := subs[0]
	end := subs[len(subs)-1]

	origin := start.String("origin")
	if o := start.Sub("origin"); o != nil {
		origin = o.Name()
	}

	destination := end.String("destination")
	if d := end.Sub("destination"); d != nil {
		destination = d.Name()
	}

	return "Trip from " + origin + " to " + destination, nil
}

func (d *DescriptionGenerator) generateRsvpDescription(ee *eagle.Entry) (string, error) {
	mm := ee.Helper()

	rsvp := mm.String(mm.TypeProperty())
	domain := util.Domain(mm.String("in-reply-to"))

	if domain == "" {
		return "", nil
	}

	switch rsvp {
	case "interested":
		return "Interested in going to an event on " + domain, nil
	case "yes":
		return "Going to an event on " + domain, nil
	case "no":
		return "Not going to an event on " + domain, nil
	case "maybe":
		return "Maybe going to an event on " + domain, nil
	}

	// TODO: leverage context information.
	return "", nil
}

func (d *DescriptionGenerator) generateWatchDescription(ee *eagle.Entry) (string, error) {
	// Matches OwnYourTrakt
	mm := ee.Helper()
	sub := mm.Sub(mm.TypeProperty())
	series := sub.Sub("episode-of")

	what := ""
	if series == nil {
		what = sub.Name()
	} else {
		season := sub.Int("season")
		episode := sub.Int("episode")
		what = sub.Name() + " (" + series.Name() + " S" + strconv.Itoa(season) + "E" + strconv.Itoa(episode) + ")"
	}

	return "Just watched: " + what, nil
}
