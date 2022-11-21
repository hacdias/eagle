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
	"github.com/samber/lo"
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

	ap.log.Debugw("received", "activity", activity)

	actor, keyID, err := ap.verifySignature(r)
	if err != nil {
		if errors.Is(err, errNotFound) {
			ap.log.Debugw("verifySignature returns not found, likely actor deleted", "key", keyID)
			// Actor has likely been deleted.
			url, err := urlpkg.Parse(keyID)
			if err == nil {
				url.Fragment = ""
				url.RawFragment = ""
				_ = ap.store.DeleteActivityPubFollower(url.String())
				return http.StatusOK, nil
			}
		}

		if errors.Is(err, errStatusUnsuccessful) {
			return http.StatusBadRequest, errors.New("could not fetch author")
		}

		return http.StatusUnauthorized, err
	}

	if activity.String("actor") != actor.String("id") {
		return http.StatusForbidden, errors.New("request actor and activity actor do not match")
	}

	ap.log.Debugw("will handle", "activity", activity, "actor", actor)

	switch activity.String("type") {
	case "Create":
		err = ap.handleCreate(r.Context(), actor, activity)
	case "Update":
		err = ap.handleUpdate(r.Context(), actor, activity)
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

func (ap *ActivityPub) createOrUpdateWebmention(ctx context.Context, actor, activity typed.Typed) error {
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

	// Activity is a reply.
	if hasReply && strings.HasPrefix(reply, ap.c.Server.BaseURL) {
		id := strings.TrimPrefix(reply, ap.c.Server.BaseURL)
		mention := ap.mentionFromActivity(actor, activity)
		mention.Type = mf2.TypeReply
		err := ap.wm.AddOrUpdateWebmention(id, mention)
		if err != nil {
			return err
		}
		return ap.store.AddActivityPubLink(id, mention.ID)
	}

	// Activity is some sort of mention.
	if hasContent && strings.Contains(content, ap.c.Server.BaseURL) {
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
			err = ap.wm.AddOrUpdateWebmention(id, mention)
			if err == nil {
				err = ap.store.AddActivityPubLink(id, mention.ID)
			}
			errs = multierror.Append(errs, err)
		}
		return errs.ErrorOrNil()
	}

	return nil
}

func (ap *ActivityPub) handleCreate(ctx context.Context, actor, activity typed.Typed) error {
	err := ap.createOrUpdateWebmention(ctx, actor, activity)
	if err != nil {
		return err
	}

	// TODO: check if I follow "actor". If so, store the activity.
	return nil
}

func (ap *ActivityPub) handleUpdate(ctx context.Context, actor, activity typed.Typed) error {
	err := ap.createOrUpdateWebmention(ctx, actor, activity)
	if err != nil {
		return err
	}

	// TODO: check if I follow "actor". If so, update the activity.
	return nil
}

func (ap *ActivityPub) handleDelete(ctx context.Context, actor, activity typed.Typed) error {
	var object string

	if objectStr, ok := activity.StringIf("object"); ok {
		object = objectStr
	} else if objectMap, ok := activity.ObjectIf("object"); ok {
		if objectStr, ok := objectMap.StringIf("id"); ok {
			object = objectStr
		} else {
			return errors.New("activity.object has no id")
		}
	} else {
		return ErrNotHandled
	}

	if object == "" {
		return errors.New("activity.object is string or map, but has no id")
	}

	entries, err := ap.store.GetActivityPubLinks(object)
	if err != nil {
		return err
	} else if len(entries) != 0 {
		// Then, it is a reply or some kind of mention.
		return ap.deleteMultipleWebmentions(entries, object)
	} else if actor.String("id") == object {
		// Otherwise, it is a user deletion.
		_ = ap.store.DeleteActivityPubFollower(object)
		return nil
	}

	return ErrNotHandled
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

	if storedInbox, err := ap.store.GetActivityPubFollower(iri); err != nil || inbox != storedInbox {
		err = ap.store.AddActivityPubFollower(iri, inbox)
		if err != nil {
			return fmt.Errorf("failed to store followers: %w", err)
		}
	}

	ap.n.Info(fmt.Sprintf("☃️ %s followed you!", iri))
	ap.SendAccept(activity, inbox)
	return nil
}

func (ap *ActivityPub) handleLikeOrAnnounce(ctx context.Context, actor, activity typed.Typed, postType mf2.Type) error {
	permalink := activity.String("object")
	if permalink == "" {
		return errors.New("activity.object is not present or is not string")
	}

	if !strings.HasPrefix(permalink, ap.c.Server.BaseURL) {
		return errors.New("activity.object is for someone else")
	}

	id := strings.TrimPrefix(permalink, ap.c.Server.BaseURL)
	mention := ap.mentionFromActivity(actor, activity)
	mention.Type = postType

	err := ap.wm.AddOrUpdateWebmention(id, mention)
	if err != nil {
		return err
	}
	return ap.store.AddActivityPubLink(id, mention.ID)
}

func (ap *ActivityPub) handleLike(ctx context.Context, actor, activity typed.Typed) error {
	return ap.handleLikeOrAnnounce(ctx, actor, activity, mf2.TypeLike)
}

func (ap *ActivityPub) handleAnnounce(ctx context.Context, actor, activity typed.Typed) error {
	return ap.handleLikeOrAnnounce(ctx, actor, activity, mf2.TypeRepost)
}

func (ap *ActivityPub) handleUndo(ctx context.Context, actor, activity typed.Typed) error {
	if object, ok := activity.StringIf("object"); ok {
		entries, err := ap.store.GetActivityPubLinks(object)
		if err != nil {
			return err
		}

		return ap.deleteMultipleWebmentions(entries, object)
	}

	if object, ok := activity.ObjectIf("object"); ok {
		switch object.String("type") {
		case "Follow":
			iri := activity.String("actor")
			if object.String("actor") != iri {
				return fmt.Errorf("%w: activity.object.actor is different from activity.actor", ErrNotHandled)
			}
			ap.n.Info(fmt.Sprintf("☃️ %s unfollowed you.", iri))
			_ = ap.store.DeleteActivityPubFollower(iri)
			return nil
		case "Like", "Announce":
			source := object.String("id")
			if source == "" {
				return fmt.Errorf("%w: activity.object.id must be string", ErrNotHandled)
			}

			permalink := object.String("object")
			if !strings.HasPrefix(permalink, ap.c.Server.BaseURL) {
				return errors.New("activity.object.object is not string or is for someone else")
			}

			id := strings.TrimPrefix(permalink, ap.c.Server.BaseURL)
			return ap.wm.DeleteWebmention(id, source)
		case "Create":
			// "Create based activities should instead use Delete, and Add activities
			// should use Remove." https://www.w3.org/TR/activitypub/#undo-activity-inbox
			return errors.New("type Create must use Delete instead of Undo")
		default:
			return ErrNotHandled
		}
	}

	return errors.New("activity.object must be string or map[string]interface{}")
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
		if id := object.String("id"); id != "" {
			// If the object has an ID, this is the ID that will be later used
			// for deleting content.
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

	links = lo.Filter(links, func(link string, _ int) bool {
		return strings.HasPrefix(link, ap.c.Server.BaseURL)
	})

	for i := range links {
		links[i] = strings.TrimPrefix(links[i], ap.c.Server.BaseURL)
	}

	return links, nil
}

func (ap *ActivityPub) deleteMultipleWebmentions(entries []string, object string) error {
	var errs *multierror.Error
	for _, entry := range entries {
		errs = multierror.Append(errs, ap.wm.DeleteWebmention(entry, object))
	}
	return errs.ErrorOrNil()
}
