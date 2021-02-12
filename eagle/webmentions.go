package eagle

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/yaml"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
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
		Author     EntryAuthor       `json:"author"`
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

type Webmentions struct {
	sync.Mutex

	domain     string
	hugoSource string
	telegraph  config.Telegraph
	media      *Media
	notify     *Notifications
	store      StorageService
	log        *zap.SugaredLogger
}

func (w *Webmentions) SendWebmention(source string, targets ...string) error {
	var errors *multierror.Error

	for _, target := range targets {
		func() {
			data := url.Values{}
			data.Set("token", w.telegraph.Token)
			data.Set("source", source)
			data.Set("target", target)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://telegraph.p3k.io/webmention", strings.NewReader(data.Encode()))
			if err != nil {
				errors = multierror.Append(errors, err)
				w.log.Errorf("error creating request: %w", err)
				return
			}

			req.Header.Set("Accept", "application/json")
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

			_, err = http.DefaultClient.Do(req)
			if err != nil {
				w.log.Warnf("could not post telegraph: %s ==> %s: %s", source, target, err)
				errors = multierror.Append(errors, err)
			}
		}()
	}

	return errors.ErrorOrNil()
}

func (w *Webmentions) ReceiveWebmentions(payload *WebmentionPayload) error {
	w.log.Infow("received webmention", "webmention", payload)

	// If it's a private notification, simply notify.
	if payload.Post.WmPrivate {
		w.notify.Notify(
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

	id, file, err := w.parseTarget(payload)
	if err != nil {
		return err
	}

	w.Lock()
	defer w.Unlock()

	mentions := []EmbeddedEntry{}
	raw, err := ioutil.ReadFile(file)
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
		newMentions := []EmbeddedEntry{}
		for _, mention := range mentions {
			if mention.URL != payload.Source {
				newMentions = append(newMentions, mention)
			}
		}

		return w.save(id, file, newMentions)
	}

	for _, mention := range mentions {
		if mention.WmID == payload.Post.WmID {
			w.log.Infof("duplicated webmention for %s: %d", id, payload.Post.WmID)
			return ErrDuplicatedWebmention
		}
	}

	ee, err := w.parsePayload(payload)
	if err != nil {
		return err
	}

	mentions = append(mentions, *ee)
	return w.save(id, file, mentions)
}

func (w *Webmentions) parseTarget(payload *WebmentionPayload) (id, file string, err error) {
	url, err := url.Parse(payload.Target)
	if err != nil {
		return "", "", err
	}

	id = url.Path
	file = strings.TrimSuffix(id, "/")
	file = strings.TrimPrefix(file, "/")
	file = strings.ReplaceAll(file, "/", "-")
	if file == "" {
		file = "index"
	}

	file = filepath.Join(w.hugoSource, "data", "interactions", file+".yaml")
	return id, file, nil
}

func (w *Webmentions) save(id, file string, mentions []EmbeddedEntry) (err error) {
	bytes, err := yaml.Marshal(mentions)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, bytes, 0644)
	if err != nil {
		return err
	}

	return w.store.Persist("webmentions: update "+id, file)
}

func (w *Webmentions) parsePayload(payload *WebmentionPayload) (*EmbeddedEntry, error) {
	ee := &EmbeddedEntry{
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
		ee.Author.Photo = w.uploadPhoto(ee.Author.Photo)
	}

	return ee, nil
}

func (w *Webmentions) uploadPhoto(url string) string {
	ext := path.Ext(url)
	base := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))

	resp, err := http.Get(url)
	if err != nil {
		w.log.Warnf("could not fetch author photo: %s", url)
		return url
	}
	defer resp.Body.Close()

	newURL, err := w.media.UploadMedia("/webmentions/"+base+ext, resp.Body)
	if err != nil {
		w.log.Errorf("could not upload photo to cdn: %s", url)
		return url
	}
	return newURL
}
