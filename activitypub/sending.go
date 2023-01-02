package activitypub

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/util"
	"github.com/hashicorp/go-multierror"
	"github.com/karlseguin/typed"
	"github.com/samber/lo"
)

func (ap *ActivityPub) sendActivity(activity typed.Typed, inboxes []string) {
	ap.log.Debugw("sending to followers", "activity", activity, "inboxes", inboxes)

	// TODO: move this to a queue that retries _n_ time in case of failures. Queue
	// handler can have a ticking time of time.Second.
	for i, inbox := range inboxes {
		if i != 0 {
			time.Sleep(time.Second)
		}

		err := ap.sendSigned(context.Background(), activity, inbox)
		if err != nil {
			ap.log.Errorw("could not send signed", "inbox", inbox, "activity", activity, "err", err)
		}
	}
}

func (ap *ActivityPub) sendActivityToFollowers(activity typed.Typed, inboxes ...string) error {
	followers, err := ap.Store.GetFollowers()
	if err != nil {
		return err
	}

	for _, f := range followers {
		inboxes = append(inboxes, f.Inbox)
	}

	inboxes = lo.Uniq(inboxes)
	go ap.sendActivity(activity, inboxes)
	return nil
}

func (ap *ActivityPub) canBePosted(e *eagle.Entry) bool {
	if e == nil {
		return false
	}

	postType := e.Helper().PostType()

	return !e.Draft &&
		!e.Deleted &&
		(lo.Contains(e.Sections, ap.Config.Site.IndexSection) ||
			postType == mf2.TypeReply ||
			postType == mf2.TypeLike ||
			postType == mf2.TypeRepost)
}

func (ap *ActivityPub) EntryHook(old, new *eagle.Entry) error {
	new, err := ap.autoLinkMentions(new)
	if err != nil {
		// Only fails if error when saving entry.
		return err
	}

	if ap.canBePosted(old) {
		if old.ID == new.ID {
			if ap.canBePosted(new) {
				return ap.sendUpdatedEntry(new)
			} else {
				return ap.sendDeletedEntry(new)
			}
		} else {
			if ap.canBePosted(new) {
				return ap.sendRenamedEntry(old, new)
			} else {
				return ap.sendDeletedEntry(old)
			}
		}
	} else {
		if ap.canBePosted(new) {
			return ap.sendNewEntry(new)
		}
	}

	return nil
}

const (
	propertyPrefix = "ap-"
)

var userMention = regexp.MustCompile(`\@\@[^\s]+\@[^\s]+\.[^\s]+`)

func (ap *ActivityPub) autoLinkMentions(e *eagle.Entry) (*eagle.Entry, error) {
	mentions := e.UserMentions

	content := userMention.ReplaceAllStringFunc(e.Content, func(s string) string {
		parts := strings.Split(strings.TrimPrefix(s, "@@"), "@")
		iri := parts[0] + "@" + parts[1]
		actor, err := ap.getActorByIRI(context.Background(), iri)
		if err == nil {
			inbox := actor.String("inbox")
			id := actor.String("id")
			if inbox != "" && id != "" {
				name := "@" + iri
				mentions = append(mentions, &eagle.UserMention{
					Name:  name,
					Href:  id,
					Inbox: inbox,
				})
				return fmt.Sprintf("[%s](%s)", name, id)
			}
		}

		return s
	})

	var (
		replyTo   string
		apReplyTo string
	)

	mm := e.Helper()
	if mm.PostType() == mf2.TypeReply {
		property := mm.TypeProperty()
		apProperty := propertyPrefix + property

		replyTo = mm.String(property)
		apReplyTo = mm.String(apProperty)

		// Do not check URLs already checked, or that are replies to self. Servers
		// such as Mastodon will only send the user a notification if they're directly
		// mentioned by a post. Therefore, we need to add a mention to the content and tags.
		// When replying to ourselves, we can ignore that.
		if replyTo != "" && apReplyTo == "" && !strings.HasPrefix(replyTo, ap.Config.Server.BaseURL) {
			actor, activity, err := ap.getActorFromActivity(context.Background(), replyTo)
			if err == nil {
				// Update apReplyTo URL if it's different from the original replyTo.
				if id := activity.String("id"); id != "" && id != replyTo {
					apReplyTo = id
				}

				// Check for the actor information.
				inbox := actor.String("inbox")
				id := actor.String("id")
				if inbox != "" && id != "" {
					found := false

					for _, m := range mentions {
						if m.Href == id {
							found = true
							break
						}
					}

					if !found {
						mentions = append(mentions, &eagle.UserMention{
							Name:  "@" + actor.String("preferredUsername") + "@" + util.Domain(id),
							Href:  id,
							Inbox: inbox,
						})
					}
				}
			}
		}
	}

	if len(mentions) == 0 {
		return e, nil
	}

	return ap.FS.TransformEntry(e.ID, func(e *eagle.Entry) (*eagle.Entry, error) {
		e.Content = content
		e.UserMentions = mentions
		if apReplyTo != "" {
			e.Properties[propertyPrefix+mm.TypeProperty()] = apReplyTo
		}
		return e, nil
	})
}

func (ap *ActivityPub) sendNewEntry(e *eagle.Entry) error {
	switch e.Helper().PostType() {
	case mf2.TypeLike:
		return ap.SendLike(e)
	case mf2.TypeRepost:
		return ap.SendAnnounce(e)
	default:
		return ap.SendCreate(e)
	}
}

func (ap *ActivityPub) sendUpdatedEntry(e *eagle.Entry) error {
	switch e.Helper().PostType() {
	case mf2.TypeLike, mf2.TypeRepost:
		// Do nothing for now. Should I do something?
		return nil
	default:
		return ap.SendUpdate(e)
	}
}

func (ap *ActivityPub) sendRenamedEntry(old, new *eagle.Entry) error {
	switch new.Helper().PostType() {
	case mf2.TypeLike, mf2.TypeRepost:
		// Do nothing for now. Should I do something?
		return nil
	default:
		return multierror.Append(ap.sendDeletedEntry(old), ap.sendNewEntry(new)).ErrorOrNil()
	}
}

func (ap *ActivityPub) sendDeletedEntry(e *eagle.Entry) error {
	switch e.Helper().PostType() {
	case mf2.TypeLike, mf2.TypeRepost:
		return ap.SendUndo(e)
	default:
		return ap.SendDelete(e.Permalink)
	}
}

func (ap *ActivityPub) SendAccept(activity typed.Typed, inbox string) {
	delete(activity, "@context")

	accept := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     "Accept",
		"id":       ap.Config.Server.BaseURL + "#" + uniuri.New(),
		"to":       activity["actor"],
		"actor":    ap.Config.Server.BaseURL,
		"object":   activity,
	}

	ap.sendActivity(accept, []string{inbox})
}

func (ap *ActivityPub) sendCreateOrUpdate(e *eagle.Entry, activityType string) error {
	object := ap.GetEntryAsActivity(e)

	var inboxes []string
	for _, mention := range e.UserMentions {
		inboxes = append(inboxes, mention.Inbox)
	}

	activity := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     activityType,
		"id":       e.Permalink,
		"to":       object["to"],
		"object":   object,
		"actor":    ap.Config.Server.BaseURL,
	}

	if cc, ok := object["cc"]; ok {
		activity["cc"] = cc
	}

	if published, ok := object["published"]; ok {
		activity["published"] = published
	}

	if updated, ok := object["updated"]; ok {
		activity["updated"] = updated
	}

	return ap.sendActivityToFollowers(activity, inboxes...)
}

func (ap *ActivityPub) SendCreate(e *eagle.Entry) error {
	return ap.sendCreateOrUpdate(e, "Create")
}

func (ap *ActivityPub) SendUpdate(e *eagle.Entry) error {
	return ap.sendCreateOrUpdate(e, "Update")
}

func (ap *ActivityPub) SendDelete(permalink string) error {
	create := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     "Delete",
		"to":       []string{"https://www.w3.org/ns/activitystreams#Public"},
		"object":   permalink,
		"actor":    ap.Config.Server.BaseURL,
	}

	return ap.sendActivityToFollowers(create)
}

func (ap *ActivityPub) sendLikeOrAnnounce(e *eagle.Entry, activityType string) error {
	mm := e.Helper()

	property := mm.TypeProperty()
	apProperty := propertyPrefix + property

	reference := mm.String(property)
	apReference := mm.String(apProperty)

	if apReference == "" {
		apReference = reference
	}

	remoteActor, remoteActivity, err := ap.getActorFromActivity(context.Background(), reference)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return nil
		} else {
			return err
		}
	}

	inbox := remoteActor.String("inbox")
	if inbox == "" {
		return nil
	}

	id := remoteActivity.String("id")
	if id != "" && id != apReference {
		apReference = id

		_, _ = ap.FS.TransformEntry(e.ID, func(e *eagle.Entry) (*eagle.Entry, error) {
			e.Properties["ap-"+property] = apReference
			return e, nil
		})
	}

	activity := map[string]interface{}{
		"@context":  []string{"https://www.w3.org/ns/activitystreams"},
		"type":      activityType,
		"id":        e.Permalink,
		"published": e.Published.Format(time.RFC3339),
		"cc": []string{
			remoteActor.String("id"),
			ap.Options.FollowersURL,
		},
		"to": []string{
			"https://www.w3.org/ns/activitystreams#Public",
		},
		"object": apReference,
		"actor":  ap.Config.Server.BaseURL,
	}

	return ap.sendActivityToFollowers(activity, inbox)
}

func (ap *ActivityPub) SendLike(e *eagle.Entry) error {
	if e.Helper().PostType() != mf2.TypeLike {
		return errors.New("post type must be like to send like")
	}

	return ap.sendLikeOrAnnounce(e, "Like")
}

func (ap *ActivityPub) SendAnnounce(e *eagle.Entry) error {
	if e.Helper().PostType() != mf2.TypeRepost {
		return errors.New("post type must be repost to send announce")
	}

	return ap.sendLikeOrAnnounce(e, "Announce")
}

func (ap *ActivityPub) SendUndo(e *eagle.Entry) error {
	if e.Helper().PostType() != mf2.TypeLike && e.Helper().PostType() != mf2.TypeRepost {
		return errors.New("can only send undo for likes and reposts")
	}

	announce := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     "Undo",
		"id":       e.Permalink + "#" + uniuri.New(),
		"to": []string{
			"https://www.w3.org/ns/activitystreams#Public",
		},
		"object": e.Permalink,
		"actor":  ap.Config.Server.BaseURL,
	}

	return ap.sendActivityToFollowers(announce)
}

func (ap *ActivityPub) SendProfileUpdate() error {
	update := map[string]any{
		"@context":  []string{"https://www.w3.org/ns/activitystreams"},
		"type":      "Update",
		"object":    ap.self,
		"actor":     ap.Config.Server.BaseURL,
		"published": time.Now().Format(time.RFC3339),
	}

	return ap.sendActivityToFollowers(update)
}
