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

func (ap *ActivityPub) InboxHandler(r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}

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
				_ = ap.Store.DeleteFollower(url.String())
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

	// TODO: Store to FS, or log to FS, https://github.com/google/uuid
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
			ap.log.Warnw("unhandled", "err", err, "activity", activity, "actor", actor)
			ap.Notifier.Info("unhandled activity")
		} else {
			ap.log.Errorw("failed", "err", err, "activity", activity, "actor", actor)
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusOK, nil
}

func (ap *ActivityPub) handleReplyOrMention(ctx context.Context, actor, activity typed.Typed) (bool, error) {
	object, err := ap.getObject(ctx, activity)
	if err != nil {
		return false, err
	}
	activity["object"] = object

	var (
		mentionType mf2.Type
		ids         []string
	)

	// Activity is a reply.
	reply := object.String("inReplyTo")
	if reply != "" && strings.HasPrefix(reply, ap.Config.Server.BaseURL) {
		mentionType = mf2.TypeReply
		id := strings.TrimPrefix(reply, ap.Config.Server.BaseURL)
		ids = append(ids, id)
	}

	// Activity is some sort of mention.
	content := object.String("content")
	if content != "" && strings.Contains(content, ap.Config.Server.BaseURL) {
		contentIDs, err := ap.getTrimmedLinksFromContent(content)
		if err == nil {
			ids = append(ids, contentIDs...)
		}
	}

	if len(ids) == 0 {
		return false, nil
	}

	mention := ap.getActivityAsMention(actor, activity)
	mention.Type = mentionType

	var errs *multierror.Error
	for _, id := range ids {
		err := ap.Webmentions.AddOrUpdateWebmention(id, mention)
		if err == nil {
			err = ap.Store.AddActivityPubLink(id, mention.ID)
		}
		errs = multierror.Append(errs, err)
	}
	return true, errs.ErrorOrNil()
}

func (ap *ActivityPub) handleMentionsTag(ctx context.Context, activity typed.Typed, isUpdate bool) bool {
	tags, ok := activity.ObjectsIf("tag")
	if !ok {
		return false
	}

	mentioned := false
	for _, tag := range tags {
		if tag.String("href") == ap.Config.Server.BaseURL {
			mentioned = true
			break
		}
	}

	if mentioned {
		if id, err := ap.getObjectID(activity); err == nil {
			if isUpdate {
				ap.Notifier.Info("✏️ Updated mention in: " + id)
			} else {
				ap.Notifier.Info("✏️ You were mentioned in: " + id)
			}
			return true
		}
	}

	return false
}

func (ap *ActivityPub) handleCreateOrUpdate(ctx context.Context, actor, activity typed.Typed, isUpdate bool) error {
	var errs *multierror.Error

	handledReplyOrMention, err := ap.handleReplyOrMention(ctx, actor, activity)
	errs = multierror.Append(errs, err)

	handledMentionsTag := ap.handleMentionsTag(ctx, activity, isUpdate)

	// TODO: check if I follow "actor". If so, store the activity.

	if err := errs.ErrorOrNil(); err != nil {
		return err
	} else if handledReplyOrMention || handledMentionsTag {
		return nil
	} else {
		return ErrNotHandled
	}
}

func (ap *ActivityPub) handleCreate(ctx context.Context, actor, activity typed.Typed) error {
	return ap.handleCreateOrUpdate(ctx, actor, activity, false)
}

func (ap *ActivityPub) handleUpdate(ctx context.Context, actor, activity typed.Typed) error {
	return ap.handleCreateOrUpdate(ctx, actor, activity, true)
}

func (ap *ActivityPub) handleDelete(ctx context.Context, actor, activity typed.Typed) error {
	object, err := ap.getObjectID(activity)
	if err != nil {
		return err
	}

	entries, err := ap.Store.GetActivityPubLinks(object)
	if err != nil {
		return err
	} else if len(entries) != 0 {
		// Then, it is a reply or some kind of mention.
		err = ap.deleteMultipleWebmentions(entries, object)
		if err == nil {
			_ = ap.Store.DeleteActivityPubLinks(object)
		}
		return err
	} else if actor.String("id") == object {
		// Otherwise, it is a user deletion.
		_ = ap.Store.DeleteFollower(object)
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

	if v, err := ap.Store.GetFollower(id); err != nil || v == nil {
		err = ap.Store.AddOrUpdateFollower(follower)
		if err != nil {
			return fmt.Errorf("failed to store followers: %w", err)
		}
	}

	ap.Notifier.Info(fmt.Sprintf("☃️ [%s](%s) followed you!", follower.Handle, follower.ID))
	ap.SendAccept(activity, inbox)
	return nil
}

func (ap *ActivityPub) handleLikeOrAnnounce(ctx context.Context, actor, activity typed.Typed, postType mf2.Type) error {
	object, err := ap.getObjectID(activity)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(object, ap.Config.Server.BaseURL) {
		return errors.New("activity.object is for someone else")
	}

	id := strings.TrimPrefix(object, ap.Config.Server.BaseURL)
	mention := ap.getActivityAsMention(actor, activity)
	mention.Type = postType

	err = ap.Webmentions.AddOrUpdateWebmention(id, mention)
	if err != nil {
		return err
	}
	return ap.Store.AddActivityPubLink(id, mention.ID)
}

func (ap *ActivityPub) handleLike(ctx context.Context, actor, activity typed.Typed) error {
	return ap.handleLikeOrAnnounce(ctx, actor, activity, mf2.TypeLike)
}

func (ap *ActivityPub) handleAnnounce(ctx context.Context, actor, activity typed.Typed) error {
	return ap.handleLikeOrAnnounce(ctx, actor, activity, mf2.TypeRepost)
}

func (ap *ActivityPub) handleUndo(ctx context.Context, actor, activity typed.Typed) error {
	if object, ok := activity.StringIf("object"); ok {
		entries, err := ap.Store.GetActivityPubLinks(object)
		if err != nil {
			return err
		}

		err = ap.deleteMultipleWebmentions(entries, object)
		if err == nil {
			_ = ap.Store.DeleteActivityPubLinks(object)
		}
		return err
	}

	if object, ok := activity.ObjectIf("object"); ok {
		switch object.String("type") {
		case "Follow":
			id := activity.String("actor")
			if object.String("actor") != id {
				return errors.New("activity.object.actor differs from activity.actor")
			}
			ap.Notifier.Info(fmt.Sprintf("☃️ %s unfollowed you.", id))
			_ = ap.Store.DeleteFollower(id)
			return nil
		case "Like", "Announce":
			source := object.String("id")
			if source == "" {
				return errors.New("activity.object.id must be string")
			}

			permalink := object.String("object")
			if !strings.HasPrefix(permalink, ap.Config.Server.BaseURL) {
				return errors.New("activity.object.object is not string or is for someone else")
			}

			id := strings.TrimPrefix(permalink, ap.Config.Server.BaseURL)
			return ap.Webmentions.DeleteWebmention(id, source)
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

func (ap *ActivityPub) getObjectID(activity typed.Typed) (string, error) {
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

func (ap *ActivityPub) getObject(ctx context.Context, activity typed.Typed) (typed.Typed, error) {
	var (
		object typed.Typed
		err    error
	)

	if objectStr := activity.String("object"); objectStr != "" {
		object, err = ap.getActivity(ctx, objectStr)
	} else if objectMap, ok := activity.ObjectIf("object"); ok {
		object = objectMap
	} else {
		return nil, errors.New("could not retrieve activity.object")
	}

	if err != nil {
		return nil, err
	}

	if id := object.String("id"); id == "" {
		return nil, errors.New("object.id is not present or is not string")
	}

	return object, nil
}

func (ap *ActivityPub) getActivityAsMention(actor, activity typed.Typed) *eagle.Mention {
	post := &eagle.Mention{
		Post: xray.Post{
			Author: ap.getActorAsXRay(actor),
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

func (ap *ActivityPub) getActorAsXRay(actor typed.Typed) xray.Author {
	author := xray.Author{
		URL:  actor.String("id"),
		Name: actor.String("name"),
	}

	icon := actor.Object("icon")
	if icon != nil {
		if icon.String("type") == "Image" {
			url := icon.String("url")

			if ap.Media != nil {
				author.Photo = ap.Media.SafeUploadFromURL("wm", url, true)
			} else {
				author.Photo = url
			}
		}
	}

	return author
}

func (ap *ActivityPub) getTrimmedLinksFromContent(body string) ([]string, error) {
	links, err := webmention.DiscoverLinksFromReader(strings.NewReader(body), "", "a")
	if err != nil {
		return nil, err
	}

	links = lo.Filter(links, func(link string, _ int) bool {
		return strings.HasPrefix(link, ap.Config.Server.BaseURL)
	})

	for i := range links {
		links[i] = strings.TrimPrefix(links[i], ap.Config.Server.BaseURL)
	}

	return links, nil
}

func (ap *ActivityPub) deleteMultipleWebmentions(entries []string, object string) error {
	var errs *multierror.Error
	for _, entry := range entries {
		errs = multierror.Append(errs, ap.Webmentions.DeleteWebmention(entry, object))
	}
	return errs.ErrorOrNil()
}
