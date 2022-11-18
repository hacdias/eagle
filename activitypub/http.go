package activitypub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/pkg/xray"
	"github.com/hashicorp/go-multierror"
	"github.com/karlseguin/typed"
	"github.com/thoas/go-funk"
	"willnorris.com/go/webmention"
)

var (
	ErrNotHandled = errors.New("request not handled")
)

func (ap *ActivityPub) HandleInbox(r *http.Request) (int, error) {
	var activity typed.Typed
	err := json.NewDecoder(r.Body).Decode(&activity)
	if err != nil {
		return http.StatusBadRequest, err
	}

	actor, keyID, err := ap.verifySignature(r)
	if err != nil {
		if errors.Is(err, errActorNotFound) {
			// Actor has likely been deleted.
			url, err := urlpkg.Parse(keyID)
			if err == nil {
				url.Fragment = ""
				url.RawFragment = ""
				_ = ap.followers.remove(url.String())
				return http.StatusOK, nil
			}
		}

		if errors.Is(err, errActorStatusUnsuccessful) {
			return http.StatusBadRequest, errors.New("could not fetch author")
		}

		return http.StatusUnauthorized, err
	}

	if activity.String("actor") != actor.String("id") {
		return http.StatusForbidden, errors.New("request actor and activity actor do not match")
	}

	switch activity.String("type") {
	case "Create":
		err = ap.handleCreate(r.Context(), actor, activity)
	case "Update":
		err = ap.handleDelete(r.Context(), actor, activity)
	case "Delete":
		err = ap.handleDelete(r.Context(), actor, activity)
	case "Follow":
		err = ap.handleFollow(r.Context(), actor, activity)
	case "Like":
		err = ap.handleLike(r.Context(), actor, activity)
	case "Announce":
		err = ap.handleAnnounce(r.Context(), actor, activity)
	case "Undo":
		err = ap.handleUndo(r.Context(), actor, activity)
	default:
		// Accept and Reject --> Answer to Follow requests
		// Add and Remove --> Weird things I did not understand
		err = ErrNotHandled
	}

	if err != nil {
		if errors.Is(err, ErrNotHandled) {
			ap.log.Infow("unhandled", "err", err, "activity", activity, "actor", actor)
			ap.n.Info("activity not handled: " + err.Error())
		} else {
			ap.log.Errorw("failed", "err", err, "activity", activity, "actor", actor)
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusOK, nil
}

func (ap *ActivityPub) handleAnnounce(ctx context.Context, actor, activity typed.Typed) error {
	permalink := activity.String("object")
	if permalink == "" {
		return fmt.Errorf("announcement object is not present or is not string")
	}

	if !strings.HasPrefix(permalink, ap.c.Server.BaseURL) {
		return fmt.Errorf("announcement destined for someone else")
	}

	id := strings.TrimPrefix(permalink, ap.c.Server.BaseURL)
	mention := ap.mentionFromActivity(actor, activity)
	mention.Type = mf2.TypeRepost
	return ap.wm.AddOrUpdateWebmention(id, mention)
}

func (ap *ActivityPub) handleFollow(ctx context.Context, actor, activity typed.Typed) error {
	iri, ok := activity.StringIf("actor")
	if !ok || len(iri) == 0 {
		return errors.New("actor not present or not string")
	}

	inbox := actor.String("inbox")
	if inbox == "" {
		return errors.New("actor has no inbox")
	}

	if storedInbox, ok := ap.followers.get(iri); !ok || inbox != storedInbox {
		err := ap.followers.set(iri, inbox)
		if err != nil {
			return fmt.Errorf("failed to store followers: %w", err)
		}
	}

	ap.n.Info(fmt.Sprintf("☃️ %s followed you!", iri))
	ap.sendAccept(activity, inbox)
	return nil
}

func (ap *ActivityPub) handleCreate(ctx context.Context, actor, activity typed.Typed) error {
	object, ok := activity.ObjectIf("object")
	if !ok {
		return fmt.Errorf("%w: object does not exist or not map", ErrNotHandled)
	}

	id, ok := object.StringIf("id")
	if !ok || len(id) == 0 {
		return fmt.Errorf("%w: object has no id", ErrNotHandled)
	}

	reply, hasReply := object.StringIf("inReplyTo")
	hasReply = hasReply && len(reply) > 0

	content, hasContent := object.StringIf("hasContent")
	hasContent = hasContent && len(content) > 0

	if hasReply && strings.HasPrefix(reply, ap.c.Server.BaseURL) {
		id := strings.TrimPrefix(reply, ap.c.Server.BaseURL)
		mention := ap.mentionFromActivity(actor, activity)
		mention.Type = mf2.TypeReply
		return ap.wm.AddOrUpdateWebmention(id, mention)
	} else if hasContent && strings.Contains(content, ap.c.Server.BaseURL) {
		mention := ap.mentionFromActivity(actor, activity)
		ids, err := ap.discoverLinksAsIDs(content)
		if err != nil {
			return err
		}

		if len(ids) == 0 {
			return ErrNotHandled
		}

		var errs *multierror.Error
		for _, id := range ids {
			errs = multierror.Append(errs, ap.wm.AddOrUpdateWebmention(id, mention))
		}
		return errs.ErrorOrNil()
	}

	return ErrNotHandled
}

func (ap *ActivityPub) handleLike(ctx context.Context, actor, activity typed.Typed) error {
	permalink := activity.String("object")
	if permalink == "" {
		return fmt.Errorf("like object is not present or is not string")
	}

	if !strings.HasPrefix(permalink, ap.c.Server.BaseURL) {
		return fmt.Errorf("like destined for someone else")
	}

	id := strings.TrimPrefix(permalink, ap.c.Server.BaseURL)
	mention := ap.mentionFromActivity(actor, activity)
	mention.Type = mf2.TypeLike
	return ap.wm.AddOrUpdateWebmention(id, mention)
}

func (ap *ActivityPub) handleDelete(ctx context.Context, actor, activity typed.Typed) error {
	object, ok := activity.StringIf("object")
	if !ok {
		return fmt.Errorf("%w: cannot handle non-string objects", ErrNotHandled)
	}

	if len(object) == 0 {
		return fmt.Errorf("%w: object is empty", ErrNotHandled)
	}

	iri := activity.String("actor")
	if iri != object {
		return fmt.Errorf("%w: object and actor differ", ErrNotHandled)
	}

	return ap.followers.remove(iri)
}

func (ap *ActivityPub) handleUndo(ctx context.Context, actor, activity typed.Typed) error {
	object, ok := activity.ObjectIf("object")
	if !ok {
		return fmt.Errorf("%w: object not present or not map: %v", ErrNotHandled, object)
	}

	switch object.String("type") {
	case "Follow":
		iri := activity.String("actor")
		if object.String("actor") != iri {
			return fmt.Errorf("%w: object actor does not match activity actor", ErrNotHandled)
		}

		return ap.followers.remove(iri)
	case "Create":
		object := activity.Object("object")
		if object == nil {
			return fmt.Errorf("%w: object is not a map", ErrNotHandled)
		}

		source := object.StringOr("id", activity.String("id"))
		if source == "" {
			return fmt.Errorf("%w: object.id is not a map", ErrNotHandled)
		}

		reply, hasReply := object.StringIf("inReplyTo")
		hasReply = hasReply && len(reply) > 0

		content, hasContent := object.StringIf("hasContent")
		hasContent = hasContent && len(content) > 0

		if hasReply && strings.HasPrefix(reply, ap.c.Server.BaseURL) {
			id := strings.TrimPrefix(reply, ap.c.Server.BaseURL)
			return ap.wm.DeleteWebmention(id, source)
		} else if hasContent && strings.Contains(content, ap.c.Server.BaseURL) {
			ids, err := ap.discoverLinksAsIDs(content)
			if err != nil {
				return err
			}

			if len(ids) == 0 {
				return ErrNotHandled
			}

			var errs *multierror.Error
			for _, id := range ids {
				errs = multierror.Append(errs, ap.wm.DeleteWebmention(id, source))
			}
			return errs.ErrorOrNil()
		}

	case "Like":
		object := activity.Object("object")
		if object == nil {
			return fmt.Errorf("%w: object is not a map", ErrNotHandled)
		}

		source := object.String("id")
		if source == "" {
			return fmt.Errorf("%w: object.id is not a map", ErrNotHandled)
		}

		permalink := object.String("object")
		if !strings.HasPrefix(permalink, ap.c.Server.BaseURL) {
			return fmt.Errorf("object.object destined for someone else")
		}

		id := strings.TrimPrefix(permalink, ap.c.Server.BaseURL)
		return ap.wm.DeleteWebmention(id, source)
	}

	return ErrNotHandled
}

func (ap *ActivityPub) mentionFromActivity(actor, activity typed.Typed) *eagle.Mention {
	post := &eagle.Mention{
		Post: xray.Post{
			Author: ap.activityActorToXray(actor),
		},
		ID: activity.String("id"),
	}

	if published := activity.String("published"); published != "" {
		t, err := dateparse.ParseStrict(published)
		if err == nil {
			post.Published = t
		}
	}

	if object := activity.Object("object"); object != nil {
		if id := object.String("id"); id != "" && post.ID == "" {
			post.ID = id
		}

		if url := object.String("url"); url != "" && post.URL == "" {
			post.URL = url
		}

		if published := object.String("published"); published != "" && post.Published.IsZero() {
			t, err := dateparse.ParseStrict(published)
			if err == nil {
				post.Published = t
			}
		}

		post.Content = xray.SanitizeContent(object.String("content"))
	}

	if post.URL == "" {
		post.URL = post.ID
	}

	return post
}

func (ap *ActivityPub) activityActorToXray(actor typed.Typed) xray.Author {
	author := xray.Author{
		URL:  actor.String("id"),
		Name: actor.String("name"),
	}

	icon := actor.Object("icon")
	if icon != nil {
		if icon.String("type") == "Image" {
			url := icon.String("url")

			if ap.media != nil {
				author.Photo = ap.media.SafeUploadFromURL("wm", url, true)
			} else {
				author.Photo = url
			}
		}
	}

	return author
}

func (ap *ActivityPub) discoverLinksAsIDs(body string) ([]string, error) {
	links, err := webmention.DiscoverLinksFromReader(strings.NewReader(body), "", "a")
	if err != nil {
		return nil, err
	}

	links = funk.FilterString(links, func(link string) bool {
		return strings.HasPrefix(link, ap.c.Server.BaseURL)
	})

	for i := range links {
		links[i] = strings.TrimPrefix(links[i], ap.c.Server.BaseURL)
	}

	return links, nil
}
