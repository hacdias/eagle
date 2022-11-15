package activitypub

import (
	"context"
	"crypto/rsa"
	"fmt"

	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/go-fed/httpsig"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/log"
	"github.com/hacdias/eagle/media"
	"github.com/hacdias/eagle/pkg/contenttype"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/renderer"
	"github.com/karlseguin/typed"
	"go.uber.org/zap"
)

type ActivityPub struct {
	c          *eagle.Config
	r          *renderer.Renderer
	fs         *fs.FS
	n          eagle.Notifier
	log        *zap.SugaredLogger
	media      *media.Media
	self       typed.Typed
	followers  *stringMapStore
	publicKey  string
	privKey    *rsa.PrivateKey
	signer     httpsig.Signer
	signerMu   sync.Mutex
	httpClient *http.Client
}

func NewActivityPub(c *eagle.Config, r *renderer.Renderer, fs *fs.FS, n eagle.Notifier, m *media.Media) (*ActivityPub, error) {
	a := &ActivityPub{
		c:     c,
		r:     r,
		fs:    fs,
		n:     n,
		media: m,
		log:   log.S().Named("activitypub"),

		httpClient: &http.Client{
			Timeout: time.Minute,
		},
	}

	var err error

	a.followers, err = newStringMapStore(filepath.Join(c.Server.ActivityPub.Directory, "followers.json"))
	if err != nil {
		return nil, err
	}

	a.privKey, a.publicKey, err = getKeyPair(c.Server.ActivityPub.Directory)
	if err != nil {
		return nil, err
	}

	a.signer, err = getSigner()
	if err != nil {
		return nil, err
	}

	a.initSelf()
	return a, nil
}

func (ap *ActivityPub) GetSelf() typed.Typed {
	return ap.self
}

func (ap *ActivityPub) GetEntry(e *eagle.Entry) typed.Typed {
	activity := map[string]interface{}{
		"@context": []string{
			"https://www.w3.org/ns/activitystreams",
		},
		"to": []string{
			"https://www.w3.org/ns/activitystreams#Public",
		},
		"id":           e.Permalink,
		"content":      string(ap.r.RenderAbsoluteMarkdown(e.Content)),
		"url":          e.Permalink,
		"mediaType":    contenttype.HTML,
		"attributedTo": ap.c.Server.BaseURL,
	}

	if e.Title != "" {
		activity["name"] = e.Title
	}

	if e.Helper().PostType() == mf2.TypeArticle {
		activity["type"] = "Article"
	} else {
		activity["type"] = "Note"
	}

	if !e.Published.IsZero() {
		activity["published"] = e.Published.Format(time.RFC3339)
	}

	if !e.Updated.IsZero() {
		activity["updated"] = e.Updated.Format(time.RFC3339)
	}

	if e.Helper().PostType() == mf2.TypeReply {
		activity["inReplyTo"] = e.Helper().String(e.Helper().TypeProperty())
	}

	for _, tag := range e.Taxonomy("tags") {
		tags := []map[string]string{}

		tags = append(tags, map[string]string{
			"type": "Hashtag",
			"name": tag,
			"id":   ap.c.Server.AbsoluteURL(fmt.Sprintf("/tags/%s", tag)),
		})

		activity["tag"] = tags
	}

	attachments := []map[string]string{}
	for _, photo := range e.Helper().Photos() {
		url := typed.Typed(photo).String("value")
		if url != "" {
			url = ap.r.GetPictureURL(url, "2000", "jpeg")
			attachments = append(attachments, map[string]string{
				"mediaType": "image/jpeg",
				"type":      "Image",
				"url":       url,
			})
		}
	}
	if len(attachments) > 0 {
		activity["attachment"] = attachments
	}

	return activity
}

func (ap *ActivityPub) initSelf() {
	self := map[string]interface{}{
		"@context": []string{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		"id":                ap.c.Server.BaseURL,
		"url":               ap.c.Server.BaseURL,
		"type":              "Person",
		"name":              ap.c.User.Name,
		"summary":           ap.c.Site.Description,
		"preferredUsername": ap.c.User.Username,
		"publicKey": map[string]interface{}{
			"id":           ap.getSelfKeyID(),
			"owner":        ap.c.Server.BaseURL,
			"publicKeyPem": ap.publicKey,
		},
		"inbox": ap.c.Server.AbsoluteURL("/activitypub/inbox"),
	}

	if ap.c.User.Photo != "" {
		self["icon"] = map[string]interface{}{
			"type":      "Image",
			"mediaType": "image/" + strings.TrimPrefix(path.Ext(ap.c.User.Photo), "."),
			"url":       ap.c.User.Photo,
		}
	}

	ap.self = self
}

func (ap *ActivityPub) getSelfKeyID() string {
	return ap.c.Server.BaseURL + "#main-key"
}

func (ap *ActivityPub) sendActivity(activity typed.Typed, inboxes []string) {
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

func (ap *ActivityPub) sendActivityToFollowers(activity typed.Typed) {
	followers := ap.followers.getAll()
	inboxes := []string{}
	for _, inbox := range followers {
		inboxes = append(inboxes, inbox)
	}
	ap.sendActivity(activity, inboxes)
}

func (ap *ActivityPub) EntryHook(e *eagle.Entry, isNew bool) error {
	if e.Listing != nil {
		return nil
	}

	activity := ap.GetEntry(e)
	if isNew {
		ap.sendCreate(activity)
	} else {
		ap.sendUpdate(activity)
	}

	if e.Helper().PostType() == mf2.TypeRead {
		ap.sendAnnounce(activity)
	}

	return nil
}

func (ap *ActivityPub) sendAccept(activity typed.Typed, inbox string) {
	delete(activity, "@context")

	accept := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     "Accept",
		"id":       ap.c.Server.BaseURL + "#" + uniuri.New(),
		"to":       activity["actor"],
		"actor":    ap.c.Server.BaseURL,
		"object":   activity,
	}

	go ap.sendActivity(accept, []string{inbox})
}

func (ap *ActivityPub) sendCreate(activity typed.Typed) {
	create := map[string]interface{}{
		"@context":  []string{"https://www.w3.org/ns/activitystreams"},
		"type":      "Create",
		"id":        activity["id"],
		"to":        activity["to"],
		"actor":     activity["attributedTo"],
		"published": activity["published"],
		"object":    activity,
	}

	go ap.sendActivityToFollowers(create)
}

func (ap *ActivityPub) sendUpdate(activity typed.Typed) {
	update := map[string]interface{}{
		"@context":  []string{"https://www.w3.org/ns/activitystreams"},
		"type":      "Update",
		"id":        activity["id"],
		"to":        activity["to"],
		"actor":     activity["attributedTo"],
		"published": activity["published"],
		"object":    activity,
	}

	if updated := activity.String("updated"); updated != "" {
		activity["updated"] = updated
	}

	go ap.sendActivityToFollowers(update)
}

func (ap *ActivityPub) sendAnnounce(activity typed.Typed) {
	announce := map[string]interface{}{
		"@context":  []string{"https://www.w3.org/ns/activitystreams"},
		"type":      "Announce",
		"id":        activity.String("id") + "#announce",
		"to":        activity["to"],
		"actor":     activity["attributedTo"],
		"published": activity["published"],
		"object":    activity,
	}

	go ap.sendActivityToFollowers(announce)
}

func isSuccess(code int) bool {
	return code == http.StatusOK ||
		code == http.StatusCreated ||
		code == http.StatusAccepted ||
		code == http.StatusNoContent
}

func isDeleted(code int) bool {
	return code == http.StatusGone ||
		code == http.StatusNotFound
}