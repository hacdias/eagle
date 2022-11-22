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
	"github.com/hacdias/eagle/util"
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
			// Actor has likely been deleted.
			ap.log.Debugw("signature not found, actor likely deleted", "activity", activity)
			url, err := urlpkg.Parse(keyID)
			if err == nil {
				url.Fragment = ""
				url.RawFragment = ""
				_ = ap.Storage.DeleteFollower(url.String())
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
		return errors.New("activity.object is not present or is not string")
	}

	id := object.String("id")
	if id == "" {
		return errors.New("activity.object.id is not present or is not string")
	}

	var (
		mentionType mf2.Type
		ids         []string
	)

	// Activity is a reply.
	reply := object.String("inReplyTo")
	if reply != "" && strings.HasPrefix(reply, ap.c.Server.BaseURL) {
		mentionType = mf2.TypeReply
		id := strings.TrimPrefix(reply, ap.c.Server.BaseURL)
		ids = append(ids, id)
	}

	// Activity is some sort of mention.
	content := object.String("content")
	if content != "" && strings.Contains(content, ap.c.Server.BaseURL) {
		contentIDs, err := ap.discoverLinksAsIDs(content)
		if err == nil {
			ids = append(ids, contentIDs...)
		}
	}

	if len(ids) == 0 {
		return nil
	}

	mention := ap.mentionFromActivity(actor, activity)
	mention.Type = mentionType

	var errs *multierror.Error
	for _, id := range ids {
		err := ap.wm.AddOrUpdateWebmention(id, mention)
		if err == nil {
			err = ap.Storage.AddActivityPubLink(id, mention.ID)
		}
		errs = multierror.Append(errs, err)
	}
	return errs.ErrorOrNil()
}

func (ap *ActivityPub) handleCreate(ctx context.Context, actor, activity typed.Typed) error {
	err := ap.createOrUpdateWebmention(ctx, actor, activity)
	if err != nil {
		return err
	}

	// multierror.Append(
	// 	ap.handleReplies(),
	// 	ap.handleMentions(),
	// 	ap.handleFollowers()
	// )

	// "tag": [
	// 		[...]
	// 		{
	// 			"href": "https://hacdias.com",
	// 			"name": "@hacdias@hacdias.com",
	// 			"type": "Mention"
	// 		}
	// 	],

	// TODO: check if I follow "actor", or if mentions IRI. If so, store the activity.
	if object, ok := activity.ObjectIf("object"); ok {
		if id := object.String("id"); id != "" {
			ap.n.Info("✏️ New Post: " + id)
			return nil
		}
	}

	return ErrNotHandled
}

func (ap *ActivityPub) handleUpdate(ctx context.Context, actor, activity typed.Typed) error {
	err := ap.createOrUpdateWebmention(ctx, actor, activity)
	if err != nil {
		return err
	}

	// TODO: check if I follow "actor", or if mentions IRI. If so, update the activity.
	if object, ok := activity.ObjectIf("object"); ok {
		if id := object.String("id"); id != "" {
			ap.n.Info("✏️ Updated Post: " + id)
			return nil
		}
	}

	return ErrNotHandled
}

func (ap *ActivityPub) getObjectAsString(activity typed.Typed) (string, error) {
	if object := activity.String("object"); object != "" {
		return object, nil
	} else if object, ok := activity.ObjectIf("object"); ok {
		if id := object.String("id"); id != "" {
			return id, nil
		}

		return "", errors.New("activity.object.id not found")
	}

	return "", errors.New("activity.object must be string or object")
}

func (ap *ActivityPub) handleDelete(ctx context.Context, actor, activity typed.Typed) error {
	object, err := ap.getObjectAsString(activity)
	if err != nil {
		return err
	}

	entries, err := ap.Storage.GetActivityPubLinks(object)
	if err != nil {
		return err
	} else if len(entries) != 0 {
		// Then, it is a reply or some kind of mention.
		return ap.deleteMultipleWebmentions(entries, object)
	} else if actor.String("id") == object {
		// Otherwise, it is a user deletion.
		_ = ap.Storage.DeleteFollower(object)
		return nil
	}

	// TODO: check if I follow "actor", or if mentions IRI. If so, delete the activity.
	return ErrNotHandled
}

func (ap *ActivityPub) handleFollow(ctx context.Context, actor, activity typed.Typed) error {
	id := activity.String("actor")
	if id == "" {
		return errors.New("actor not present or not string")
	}

	inbox := actor.String("inbox")
	if inbox == "" {
		return errors.New("actor has no inbox")
	}

	follower := Follower{
		Name:   actor.String("name"),
		ID:     id,
		Inbox:  inbox,
		Handle: fmt.Sprintf("@%s@%s", actor.String("preferredUsername"), util.Domain(id)),
	}

	if _, err := ap.Storage.GetFollower(id); err != nil {
		err = ap.Storage.AddOrUpdateFollower(follower)
		if err != nil {
			return fmt.Errorf("failed to store followers: %w", err)
		}
	}

	ap.n.Info(fmt.Sprintf("☃️ %s (%s) followed you!", follower.Handle, follower.ID))
	ap.SendAccept(activity, inbox)
	return nil
}

func (ap *ActivityPub) handleLikeOrAnnounce(ctx context.Context, actor, activity typed.Typed, postType mf2.Type) error {
	object, err := ap.getObjectAsString(activity)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(object, ap.c.Server.BaseURL) {
		return errors.New("activity.object is for someone else")
	}

	id := strings.TrimPrefix(object, ap.c.Server.BaseURL)
	mention := ap.mentionFromActivity(actor, activity)
	mention.Type = postType

	err = ap.wm.AddOrUpdateWebmention(id, mention)
	if err != nil {
		return err
	}
	return ap.Storage.AddActivityPubLink(id, mention.ID)
}

func (ap *ActivityPub) handleLike(ctx context.Context, actor, activity typed.Typed) error {
	return ap.handleLikeOrAnnounce(ctx, actor, activity, mf2.TypeLike)
}

func (ap *ActivityPub) handleAnnounce(ctx context.Context, actor, activity typed.Typed) error {
	return ap.handleLikeOrAnnounce(ctx, actor, activity, mf2.TypeRepost)
}

func (ap *ActivityPub) handleUndo(ctx context.Context, actor, activity typed.Typed) error {
	if object, ok := activity.StringIf("object"); ok {
		entries, err := ap.Storage.GetActivityPubLinks(object)
		if err != nil {
			return err
		}

		return ap.deleteMultipleWebmentions(entries, object)
	}

	if object, ok := activity.ObjectIf("object"); ok {
		switch object.String("type") {
		case "Follow":
			id := activity.String("actor")
			if object.String("actor") != id {
				return errors.New("activity.object.actor differs from activity.actor")
			}
			ap.n.Info(fmt.Sprintf("☃️ %s unfollowed you.", id))
			_ = ap.Storage.DeleteFollower(id)
			return nil
		case "Like", "Announce":
			source := object.String("id")
			if source == "" {
				return errors.New("activity.object.id must be string")
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
