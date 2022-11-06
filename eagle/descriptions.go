package eagle

import (
	"strconv"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/entry/mf2"
)

var typeToDescription = map[mf2.Type]string{
	mf2.TypeReply:    "Replied to ",
	mf2.TypeLike:     "Liked ",
	mf2.TypeRepost:   "Reposted ",
	mf2.TypeBookmark: "Bookmarked ",
	mf2.TypeAte:      "Just ate: ",
	mf2.TypeDrank:    "Just drank: ",
}

func (e *Eagle) GenerateDescription(ee *entry.Entry, force bool) error {
	if ee.Description != "" && !force {
		return nil
	}

	var (
		description string
		err         error
	)

	mm := ee.Helper()

	switch mm.PostType() {
	case mf2.TypeReply,
		mf2.TypeLike,
		mf2.TypeRepost,
		mf2.TypeBookmark:
		url := mm.String(mm.TypeProperty())
		urlDomain := domain(url)
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
		description, err = e.generateItineraryDescription(ee)
	case mf2.TypeRsvp:
		description, err = e.generateRsvpDescription(ee)
	case mf2.TypeWatch:
		description, err = e.generateWatchDescription(ee)
	}

	if err != nil {
		return err
	}

	if description == "" && ee.Description != "" {
		return nil
	}

	ee.Description = description
	return nil
}

func (e *Eagle) generateItineraryDescription(ee *entry.Entry) (string, error) {
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

func (e *Eagle) generateRsvpDescription(ee *entry.Entry) (string, error) {
	mm := ee.Helper()

	rsvp := mm.String(mm.TypeProperty())
	domain := domain(mm.String("in-reply-to"))

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

func (e *Eagle) generateWatchDescription(ee *entry.Entry) (string, error) {
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
