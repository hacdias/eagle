package eagle

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/yaml"
	"github.com/hashicorp/go-multierror"
	"willnorris.com/go/webmention"
)

var ErrDuplicatedWebmention = errors.New("duplicated webmention")

var webmentionTypes = map[string]string{
	"like-of":     "like",
	"repost-of":   "repost",
	"mention-of":  "mention",
	"in-reply-to": "reply",
}

type WebmentionPayload struct {
	Secret  string `json:"secret"`
	Source  string `json:"source"`
	Deleted bool   `json:"deleted"`
	Target  string `json:"target"`
	Post    struct {
		Type       string            `json:"type"`
		Author     Author            `json:"author"`
		URL        string            `json:"url"`
		Published  string            `json:"published"`
		WmReceived string            `json:"wm-received"`
		WmID       int               `json:"wm-id"`
		Content    WebmentionContent `json:"content"`
		MentionOf  string            `json:"mention-of"`
		WmProperty string            `json:"wm-property"`
		WmSource   string            `json:"wm-source"`
		WmPrivate  bool              `json:"wm-private"`
	} `json:"post"`
}

type WebmentionContent struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

func (e *Eagle) SendWebmention(source string, targets ...string) error {
	var errs *multierror.Error

	for _, target := range targets {
		err := e.sendWebmention(source, target)
		if err != nil && !errors.Is(err, webmention.ErrNoEndpointFound) {
			err = fmt.Errorf("webmention error %s: %w", target, err)
			errs = multierror.Append(errs, err)
		}
	}

	return errs.ErrorOrNil()
}

func (e *Eagle) sendWebmention(source, target string) error {
	endpoint, err := e.webmentionsClient.DiscoverEndpoint(target)
	if err != nil {
		return err
	}

	res, err := e.webmentionsClient.SendWebmention(endpoint, source, target)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, res.Body)
	defer res.Body.Close()

	return nil
}

func (e *Eagle) ReceiveWebmentions(payload *WebmentionPayload) error {
	e.log.Infow("received webmention", "webmention", payload)

	// If it's a private notification, simply notify.
	if payload.Post.WmPrivate {
		e.Notify(
			fmt.Sprintf(
				"Received private webmention from %s at %s to %s: %s",
				payload.Post.Author.Name,
				payload.Post.URL,
				payload.Target,
				payload.Post.Content.Text,
			),
		)
		return nil
	}

	id, file, err := e.parseWebmentionTarget(payload)
	if err != nil {
		return err
	}

	e.webmentionsMu.Lock()
	defer e.webmentionsMu.Unlock()

	mentions := []XRay{}
	raw, err := e.ReadFile(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	err = yaml.Unmarshal(raw, &mentions)
	if err != nil {
		return err
	}

	if payload.Deleted {
		newMentions := []XRay{}
		for _, mention := range mentions {
			if mention.URL != payload.Source {
				newMentions = append(newMentions, mention)
			}
		}

		return e.saveWebmentions(id, file, newMentions)
	}

	for _, mention := range mentions {
		if mention.WmID == payload.Post.WmID {
			e.log.Infof("duplicated webmention for %s: %d", id, payload.Post.WmID)
			return ErrDuplicatedWebmention
		}
	}

	ee, err := e.parseWebmentionPayload(payload)
	if err != nil {
		return err
	}

	mentions = append(mentions, *ee)
	return e.saveWebmentions(id, file, mentions)
}

func (e *Eagle) parseWebmentionTarget(payload *WebmentionPayload) (id, file string, err error) {
	url, err := urlpkg.Parse(payload.Target)
	if err != nil {
		return "", "", err
	}

	id = filepath.Clean(url.Path)
	dir := filepath.Join("content", id)

	if stat, err := e.srcFs.Stat(dir); err != nil || !stat.IsDir() {
		if !stat.IsDir() {
			err = fmt.Errorf("entry is not a bundle")
		}
		return id, file, err
	}

	file = filepath.Join(dir, "interactions.yaml")

	return id, file, nil
}

func (e *Eagle) parseWebmentionPayload(payload *WebmentionPayload) (*XRay, error) {
	ee := &XRay{
		WmID:   payload.Post.WmID,
		Author: &payload.Post.Author,
	}

	if payload.Post.Content.Text != "" {
		ee.Content = payload.Post.Content.Text
	} else if payload.Post.Content.HTML != "" {
		ee.Content = payload.Post.Content.HTML
	}

	if payload.Post.WmProperty != "" {
		if v, ok := webmentionTypes[payload.Post.WmProperty]; ok {
			ee.Type = v
		} else {
			ee.Type = "mention"
		}
	} else {
		ee.Type = "mention"
	}

	if payload.Post.URL != "" {
		ee.URL = payload.Post.URL
	} else {
		ee.URL = payload.Post.WmSource
	}

	var err error
	if payload.Post.Published != "" {
		ee.Date, err = dateparse.ParseStrict(payload.Post.Published)
	} else {
		ee.Date, err = dateparse.ParseStrict(payload.Post.WmReceived)
	}
	if err != nil {
		return nil, err
	}

	if ee.Author.Photo != "" {
		ee.Author.Photo = e.uploadWebmentionPhoto(ee.Author.Photo)
	}

	return ee, nil
}

func (e *Eagle) saveWebmentions(id, file string, mentions []XRay) (err error) {
	bytes, err := yaml.Marshal(mentions)
	if err != nil {
		return err
	}

	return e.Persist(file, bytes, "webmentions: update "+id)
}

func (e *Eagle) uploadWebmentionPhoto(url string) string {
	if e.media == nil {
		return url
	}

	ext := path.Ext(url)
	base := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))

	resp, err := http.Get(url)
	if err != nil {
		e.log.Warnf("could not fetch author photo: %s", url)
		return url
	}
	defer resp.Body.Close()

	newURL, err := e.media.UploadMedia("/wm/"+base+ext, resp.Body)
	if err != nil {
		e.log.Errorf("could not upload photo to cdn: %s", url)
		return url
	}
	return newURL
}
