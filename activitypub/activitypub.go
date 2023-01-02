// ActivityPub implementation https://www.w3.org/TR/activitypub/
package activitypub

import (
	"bytes"
	"crypto/rsa"
	"errors"
	"fmt"
	"mime"

	"net/http"
	urlpkg "net/url"
	"path"
	"sync"
	"time"

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
	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"
)

type Follower struct {
	Name   string
	ID     string
	Inbox  string
	Handle string
}

type FollowersStorage interface {
	AddOrUpdateFollower(Follower) error
	GetFollower(id string) (*Follower, error)
	GetFollowers() ([]*Follower, error)
	GetFollowersByPage(page, limit int) ([]*Follower, error)
	GetFollowersCount() (int, error)
	DeleteFollower(iri string) error
}

type LinksStorage interface {
	AddActivityPubLink(entry, activity string) error
	GetActivityPubLinks(activity string) ([]string, error)
	DeleteActivityPubLinks(activity string) error
}

type Storage interface {
	FollowersStorage
	LinksStorage
}

type Options struct {
	Config      *eagle.Config
	Renderer    *renderer.Renderer
	FS          *fs.FS
	Notifier    eagle.Notifier
	Webmentions *webmentions.Webmentions
	Store       Storage

	// Optional fields.
	Media        *media.Media
	InboxURL     string
	OutboxURL    string
	FollowersURL string
}

type ActivityPub struct {
	*Options

	log        *zap.SugaredLogger
	httpClient *http.Client

	publicKey  string
	privateKey *rsa.PrivateKey
	account    string // username@domain.org
	self       typed.Typed

	signerMu sync.Mutex
	signer   httpsig.Signer
}

func NewActivityPub(options *Options) (*ActivityPub, error) {
	if options.Config == nil || options.Renderer == nil ||
		options.FS == nil || options.Notifier == nil ||
		options.Webmentions == nil || options.Store == nil {
		return nil, errors.New("required fields in Options are missing")
	}

	url, _ := urlpkg.Parse(options.Config.Server.BaseURL)

	a := &ActivityPub{
		Options: options,
		log:     log.S().Named("activitypub"),
		httpClient: &http.Client{
			Timeout: time.Minute,
		},
		account: fmt.Sprintf("%s@%s", options.Config.User.Username, url.Host),
	}

	var err error

	a.privateKey, a.publicKey, err = getKeyPair(a.Config.Server.ActivityPub.Directory)
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

func (ap *ActivityPub) GetEntryAsActivity(e *eagle.Entry) typed.Typed {
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
		"attributedTo": ap.Config.Server.BaseURL,
	}

	var (
		buf     bytes.Buffer
		content string
	)
	err := ap.Renderer.Render(&buf, &renderer.RenderData{Entry: e}, []string{renderer.TemplateActivityPub}, true)
	if err != nil {
		content = string(ap.Renderer.RenderAbsoluteMarkdown(e.Content))
	} else {
		content = buf.String()
	}

	if content != "" {
		activity["content"] = bluemonday.UGCPolicy().Sanitize(content)
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
		mm := e.Helper()
		property := mm.TypeProperty()
		apProperty := propertyPrefix + e.Helper().TypeProperty()
		activity["inReplyTo"] = mm.Properties.StringOr(apProperty, mm.String(property))
	}

	tags := []map[string]string{}

	if ap.Config.Server.ActivityPub.TagTaxonomy != "" {
		for _, tag := range e.Taxonomy(ap.Config.Server.ActivityPub.TagTaxonomy) {
			tags = append(tags, map[string]string{
				"type": "Hashtag",
				"name": tag,
				"id":   ap.Config.Server.AbsoluteURL(fmt.Sprintf("/%s/%s", ap.Config.Server.ActivityPub.TagTaxonomy, tag)),
			})
		}
	}

	for _, mention := range e.UserMentions {
		tags = append(tags, map[string]string{
			"type": "Mention",
			"name": mention.Name,
			"href": mention.Href,
			"id":   mention.Href,
		})
	}

	if len(tags) > 0 {
		activity["tag"] = tags
	}

	attachments := []map[string]string{}
	for _, photo := range e.Helper().Photos() {
		url := typed.Typed(photo).String("value")
		if url != "" {
			url = ap.Renderer.ResolveImageURL(url)
			attachments = append(attachments, mediaToActivity(url, "Image"))
		}
	}

	for _, video := range e.Helper().Videos() {
		url := typed.Typed(video).String("value")
		if url != "" {
			attachments = append(attachments, mediaToActivity(url, "Video"))
		}
	}

	for _, audio := range e.Helper().Audios() {
		url := typed.Typed(audio).String("value")
		if url != "" {
			attachments = append(attachments, mediaToActivity(url, "Audio"))
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
		"id":                ap.Config.Server.BaseURL,
		"url":               ap.Config.Server.BaseURL,
		"type":              "Person",
		"name":              ap.Config.User.Name,
		"summary":           ap.Config.Site.Description,
		"preferredUsername": ap.Config.User.Username,
		"publicKey": map[string]interface{}{
			"id":           ap.getSelfKeyID(),
			"owner":        ap.Config.Server.BaseURL,
			"publicKeyPem": ap.publicKey,
		},
		"inbox":  ap.Options.InboxURL,
		"outbox": ap.Options.OutboxURL,
	}

	if ap.Config.User.Photo != "" {
		self["icon"] = mediaToActivity(ap.Config.User.Photo, "Image")
	}

	if ap.Config.User.CoverPhoto != "" {
		self["image"] = mediaToActivity(ap.Config.User.CoverPhoto, "Image")
	}

	if !ap.Config.User.Published.IsZero() {
		self["published"] = ap.Config.User.Published.Format(time.RFC3339)
	}

	attachments := []map[string]string{
		linkToActivity(ap.Config.Server.ActivityPub.WebsitePropertyName, ap.Config.Server.BaseURL),
	}

	self["attachment"] = attachments
	ap.self = self
}

func (ap *ActivityPub) getSelfKeyID() string {
	return ap.Config.Server.BaseURL + "#main-key"
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

func mediaToActivity(url, typ string) map[string]string {
	return map[string]string{
		"type":      typ,
		"mediaType": mime.TypeByExtension(path.Ext(url)),
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
