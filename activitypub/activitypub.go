package activitypub

import (
	"bytes"
	"context"
	"crypto/rsa"
	"fmt"
	"mime"

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
	"github.com/hacdias/eagle/util"
	"github.com/hacdias/eagle/webmentions"
	"github.com/karlseguin/typed"
	"github.com/thoas/go-funk"
	"go.uber.org/zap"
)

type ActivityPub struct {
	c          *eagle.Config
	r          *renderer.Renderer
	fs         *fs.FS
	n          eagle.Notifier
	wm         *webmentions.Webmentions
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

func NewActivityPub(c *eagle.Config, r *renderer.Renderer, fs *fs.FS, n eagle.Notifier, wm *webmentions.Webmentions, m *media.Media) (*ActivityPub, error) {
	a := &ActivityPub{
		c:     c,
		r:     r,
		fs:    fs,
		n:     n,
		media: m,
		wm:    wm,
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
		"url":          e.Permalink,
		"mediaType":    contenttype.HTML,
		"attributedTo": ap.c.Server.BaseURL,
	}

	var buf bytes.Buffer
	err := ap.r.Render(&buf, &renderer.RenderData{Entry: e}, []string{renderer.TemplateActivityPub})
	if err != nil {
		activity["content"] = string(ap.r.RenderAbsoluteMarkdown(e.Content))
	} else {
		activity["content"] = buf.String()
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

	if ap.c.Server.ActivityPub.TagTaxonomy != "" {
		tags := []map[string]string{}
		for _, tag := range e.Taxonomy(ap.c.Server.ActivityPub.TagTaxonomy) {
			tags = append(tags, map[string]string{
				"type": "Hashtag",
				"name": tag,
				"id":   ap.c.Server.AbsoluteURL(fmt.Sprintf("/%s/%s", ap.c.Server.ActivityPub.TagTaxonomy, tag)),
			})
		}
		if len(tags) > 0 {
			activity["tag"] = tags
		}
	}

	attachments := []map[string]string{}
	for _, photo := range e.Helper().Photos() {
		url := typed.Typed(photo).String("value")
		if url != "" {
			url = ap.r.GetPictureURL(url, "2000", "jpeg")
			attachments = append(attachments, imageToActivity(url))
		}

		// TODO: add videos and audios.
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
		"inbox":  ap.c.Server.AbsoluteURL("/activitypub/inbox"),
		"outbox": ap.c.Server.AbsoluteURL("/activitypub/outbox"),
	}

	if ap.c.User.Photo != "" {
		self["icon"] = imageToActivity(ap.c.User.Photo)
	}

	if ap.c.User.CoverPhoto != "" {
		self["image"] = imageToActivity(ap.c.User.CoverPhoto)
	}

	if !ap.c.User.Published.IsZero() {
		self["published"] = ap.c.User.Published.Format(time.RFC3339)
	}

	attachments := []map[string]string{
		linkToActivity(ap.c.Server.ActivityPub.WebsitePropertyName, ap.c.Server.BaseURL),
	}

	self["attachment"] = attachments
	ap.self = self
}

func (ap *ActivityPub) getSelfKeyID() string {
	return ap.c.Server.BaseURL + "#main-key"
}

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

func (ap *ActivityPub) sendActivityToFollowers(activity typed.Typed) {
	followers := ap.followers.getAll()
	inboxes := []string{}
	for _, inbox := range followers {
		inboxes = append(inboxes, inbox)
	}
	ap.sendActivity(activity, inboxes)
}

func (ap *ActivityPub) canBePosted(e *eagle.Entry) bool {
	if e == nil {
		return false
	}

	return !e.Draft &&
		!e.Deleted &&
		funk.ContainsString(e.Sections, ap.c.Site.IndexSection) &&
		e.Visibility() == eagle.VisibilityPublic
}

func (ap *ActivityPub) EntryHook(old, new *eagle.Entry) error {
	if ap.canBePosted(old) {
		if !ap.canBePosted(new) {
			ap.SendDelete(new.Permalink)
		} else if old.ID != new.ID {
			ap.SendDelete(old.Permalink)
			ap.SendCreate(new)
		} else {
			ap.SendUpdate(new)
		}
	} else {
		if ap.canBePosted(new) {
			ap.SendCreate(new)

			if new.Helper().PostType() == mf2.TypeRead {
				ap.SendAnnounce(new)
			}

			return nil
		}
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

func (ap *ActivityPub) SendCreate(e *eagle.Entry) {
	activity := ap.GetEntry(e)

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

	go ap.sendActivityToFollowers(create)
}

func (ap *ActivityPub) SendUpdate(e *eagle.Entry) {
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

	go ap.sendActivityToFollowers(update)
}

func (ap *ActivityPub) SendDelete(permalink string) {
	create := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     "Delete",
		"to":       []string{"https://www.w3.org/ns/activitystreams#Public"},
		"object":   permalink,
		"actor":    ap.c.Server.BaseURL,
	}

	go ap.sendActivityToFollowers(create)
}

func (ap *ActivityPub) SendAnnounce(e *eagle.Entry) {
	activity := ap.GetEntry(e)

	announce := map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/activitystreams"},
		"type":     "Announce",
		"id":       activity.String("id") + "#announce",
		"to":       activity["to"],
		"object":   activity,
		"actor":    ap.c.Server.BaseURL,
	}

	if published, ok := activity["published"]; ok {
		announce["published"] = published
	}

	if updated, ok := activity["updated"]; ok {
		announce["updated"] = updated
	}

	go ap.sendActivityToFollowers(announce)
}

func (ap *ActivityPub) SendProfileUpdate() {
	update := map[string]any{
		"@context":  []string{"https://www.w3.org/ns/activitystreams"},
		"type":      "Update",
		"object":    ap.self,
		"actor":     ap.c.Server.BaseURL,
		"published": time.Now().Format(time.RFC3339),
	}

	go ap.sendActivityToFollowers(update)
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

func imageToActivity(url string) map[string]string {
	ext := path.Ext(url)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		if ext == ".jpg" {
			mimeType = "image/jpeg"
		} else {
			mimeType = "image/" + strings.TrimPrefix(ext, ".")
		}
	}

	return map[string]string{
		"type":      "Image",
		"mediaType": mimeType,
		"url":       url,
	}
}

func linkToActivity(name, url string) map[string]string {
	return map[string]string{
		"type":  "PropertyValue",
		"name":  name,
		"value": fmt.Sprintf(`<a href="%s">%s</a>`, url, util.StripScheme(url)),
	}
}
