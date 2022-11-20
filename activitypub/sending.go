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
	followers, err := ap.store.GetActivityPubFollowers()
	if err != nil {
		return err
	}

	for _, inbox := range followers {
		inboxes = append(inboxes, inbox)
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
		e.Visibility() != eagle.VisibilityPrivate &&
		(lo.Contains(e.Sections, ap.c.Site.IndexSection) ||
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

var userMention = regexp.MustCompile(`\@\@[^\s]+\@[^\s]+\.[^\s]+`)

func (ap *ActivityPub) autoLinkMentions(e *eagle.Entry) (*eagle.Entry, error) {
	var mentions []*eagle.UserMention

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

	if e.Helper().PostType() == mf2.TypeReply {
		replyTo := e.Helper().String(e.Helper().TypeProperty())
		if replyTo != "" {
			actor, err := ap.getActorFromActivity(context.Background(), replyTo)
			if err == nil {
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
						name := "@" + actor.String("preferredUsername") + "@" + util.Domain(id)
						mentions = append(mentions, &eagle.UserMention{
							Name:  name,
							Href:  id,
							Inbox: inbox,
						})
						content = fmt.Sprintf("[%s](%s) ", name, id) + content
					}
				}
			}
		}
	}

	if len(mentions) == 0 {
		return e, nil
	}

	return ap.fs.TransformEntry(e.ID, func(e *eagle.Entry) (*eagle.Entry, error) {
		e.Content = content
		e.UserMentions = append(e.UserMentions, mentions...)
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
		"id":       ap.c.Server.BaseURL + "#" + uniuri.New(),
		"to":       activity["actor"],
		"actor":    ap.c.Server.BaseURL,
		"object":   activity,
	}

	ap.sendActivity(accept, []string{inbox})
}

func (ap *ActivityPub) SendCreate(e *eagle.Entry) error {
	activity := ap.GetEntry(e)

	var inboxes []string
	for _, mention := range e.UserMentions {
		inboxes = append(inboxes, mention.Inbox)
	}

	create := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     "Create",
		"id":       e.Permalink,
		"to":       activity["to"],
		"object":   activity,
		"actor":    ap.c.Server.BaseURL,
	}

	if published, ok := activity["published"]; ok {
		create["published"] = published
	}

	return ap.sendActivityToFollowers(create, inboxes...)
}

func (ap *ActivityPub) SendUpdate(e *eagle.Entry) error {
	activity := ap.GetEntry(e)

	update := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     "Update",
		"id":       activity["id"],
		"to":       activity["to"],
		"object":   activity,
		"actor":    ap.c.Server.BaseURL,
	}

	if published, ok := activity["published"]; ok {
		update["published"] = published
	}

	if updated, ok := activity["updated"]; ok {
		update["updated"] = updated
	}

	return ap.sendActivityToFollowers(update)
}

func (ap *ActivityPub) SendDelete(permalink string) error {
	create := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     "Delete",
		"to":       []string{"https://www.w3.org/ns/activitystreams#Public"},
		"object":   permalink,
		"actor":    ap.c.Server.BaseURL,
	}

	return ap.sendActivityToFollowers(create)
}

func (ap *ActivityPub) sendLikeOrAnnounce(e *eagle.Entry, activityType string) error {
	target := e.Helper().String(e.Helper().TypeProperty())
	actor, err := ap.getActorFromActivity(context.Background(), target)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return nil
		} else {
			return err
		}
	}

	inbox := actor.String("inbox")
	if len(inbox) == 0 {
		return nil
	}

	activity := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     activityType,
		"id":       e.Permalink,
		"to": []string{
			"https://www.w3.org/ns/activitystreams#Public",
		},
		"object": e.Helper().String(e.Helper().TypeProperty()),
		"actor":  ap.c.Server.BaseURL,
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
		"object": e.Helper().String(e.Helper().TypeProperty()),
		"actor":  ap.c.Server.BaseURL,
	}

	return ap.sendActivityToFollowers(announce)
}

func (ap *ActivityPub) SendProfileUpdate() error {
	update := map[string]any{
		"@context":  []string{"https://www.w3.org/ns/activitystreams"},
		"type":      "Update",
		"object":    ap.self,
		"actor":     ap.c.Server.BaseURL,
		"published": time.Now().Format(time.RFC3339),
	}

	return ap.sendActivityToFollowers(update)
}
