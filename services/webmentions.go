package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
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

type WebmentionContent struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

type WebmentionAuthor struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Photo string `json:"photo"`
}

type WebmentionPayload struct {
	Secret  string `json:"secret"`
	Source  string `json:"source"`
	Deleted bool   `json:"deleted"`
	Target  string `json:"target"`
	Post    struct {
		Type       string            `json:"type"`
		Author     WebmentionAuthor  `json:"author"`
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

type Webmentions struct {
	*zap.SugaredLogger
	Domain    string
	Telegraph config.Telegraph
	Media     *Media
	Hugo      *Hugo
}

func (w *Webmentions) Send(source string, targets ...string) error {
	var errors *multierror.Error

	for _, target := range targets {
		func() {
			data := url.Values{}
			data.Set("token", w.Telegraph.Token)
			data.Set("source", source)
			data.Set("target", target)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://telegraph.p3k.io/webmention", strings.NewReader(data.Encode()))
			if err != nil {
				errors = multierror.Append(errors, err)
				w.Errorf("error creating request: %s", err)
				return
			}

			req.Header.Set("Accept", "application/json")
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

			_, err = http.DefaultClient.Do(req)
			if err != nil {
				w.Warnf("could not post telegraph: %s ==> %s: %s", source, target, err)
				errors = multierror.Append(errors, err)
			}
		}()
	}

	return errors.ErrorOrNil()
}

func (w *Webmentions) Receive(payload *WebmentionPayload) error {
	permalink := strings.Replace(payload.Target, w.Domain, "", 1)
	storeFile := path.Join(w.Hugo.Source, "content", permalink, "mentions.json")

	if _, err := os.Stat(path.Join(w.Hugo.Source, "content", permalink)); os.IsNotExist(err) {
		storeFile = path.Join(w.Hugo.Source, "data", "mentions", "orphans.json")
	} else if payload.Post.WmPrivate {
		storeFile = path.Join(w.Hugo.Source, "data", "mentions", "private.json")
	}

	mentions := []StoredWebmention{}

	if fd, err := os.Open(storeFile); err == nil {
		err := json.NewDecoder(fd).Decode(&mentions)
		if err != nil {
			fd.Close()
			return err
		}
		fd.Close()
	}

	if payload.Deleted {
		newMentions := []StoredWebmention{}
		for _, mention := range mentions {
			if mention.URL != payload.Source {
				newMentions = append(newMentions, mention)
			}
		}

		return w.save(newMentions, storeFile)
	}

	for _, mention := range mentions {
		if mention.ID == payload.Post.WmID {
			w.Infof("duplicated webmention for %s: %d", permalink, payload.Post.WmID)
			return ErrDuplicatedWebmention
		}
	}

	wm := &StoredWebmention{
		ID:      payload.Post.WmID,
		Content: payload.Post.Content,
		Author:  payload.Post.Author,
	}

	if payload.Post.WmProperty != "" {
		if v, ok := webmentionTypes[payload.Post.WmProperty]; ok {
			wm.Type = v
		} else {
			wm.Type = "mention"
		}
	} else {
		wm.Type = "mention"
	}

	if payload.Post.URL != "" {
		wm.URL = payload.Post.URL
	} else {
		wm.URL = payload.Post.WmSource
	}

	if payload.Post.Published != "" {
		wm.Date = payload.Post.Published
	} else {
		wm.Date = payload.Post.WmReceived
	}

	if wm.Author.Photo != "" {
		wm.Author.Photo = w.uploadPhoto(wm.Author.Photo)
	}

	mentions = append(mentions, *wm)
	return w.save(mentions, storeFile)
}

func (w *Webmentions) save(mentions []StoredWebmention, file string) error {
	bytes, err := json.MarshalIndent(mentions, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(file, bytes, 0644)
}

type StoredWebmention struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Date    string            `json:"date"`
	ID      int               `json:"wm-id"`
	Content WebmentionContent `json:"content"`
	Author  WebmentionAuthor  `json:"author"`
}

func (w *Webmentions) uploadPhoto(url string) string {
	ext := path.Ext(url)
	base := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))

	resp, err := http.Get(url)
	if err != nil {
		w.Warnf("could not fetch author photo: %s", url)
		return url
	}
	defer resp.Body.Close()

	newURL, err := w.Media.UploadMedia("/webmentions/"+base+ext, resp.Body)
	if err != nil {
		w.Errorf("could not upload photo to cdn: %s", url)
		return url
	}
	return newURL
}
