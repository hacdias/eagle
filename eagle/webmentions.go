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
	"strings"

	"github.com/araddon/dateparse"
	"github.com/hashicorp/go-multierror"
	"willnorris.com/go/webmention"
)

var webmentionTypes = map[string]string{
	"like-of":     "like",
	"repost-of":   "repost",
	"mention-of":  "mention",
	"in-reply-to": "reply",
	"bookmark-of": "bookmark",
	"rsvp":        "rsvp",
}

type Webmention struct {
	XRay `yaml:",inline"`
	// Specifically for webmentions received from https://webmention.io
	// TODO: remove this and compare webmentions via URL.
	WmID    int  `yaml:"wm-id,omitempty" json:"wm-id,omitempty"`
	Private bool `json:"private,omitempty"`
}

type WebmentionPayload struct {
	Source  string `json:"source"`
	Secret  string `json:"secret"`
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

func (e *Eagle) SendWebmentions(entry *Entry) error {
	all, curr, _, err := e.GetWebmentionTargets(entry)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	for _, target := range all {
		if strings.HasPrefix(target, e.Config.BaseURL) {
			// TODO: it is a self-mention
			e.log.Infof("TODO: self-mention from %s to %s", entry.Permalink, target)
		}

		err := e.sendWebmention(entry.Permalink, target)
		if err != nil && !errors.Is(err, webmention.ErrNoEndpointFound) {
			err = fmt.Errorf("webmention error %s: %w", target, err)
			errs = multierror.Append(errs, err)
		}
	}

	if !entry.Deleted {
		// If it's not a deleted entry, update the targets list.
		err = e.TransformEntryData(entry, func(data *EntryData) (*EntryData, error) {
			data.Targets = curr
			return data, nil
		})

		errs = multierror.Append(errs, err)
	}

	return errs.ErrorOrNil()
}

func (e *Eagle) GetWebmentionTargets(entry *Entry) ([]string, []string, []string, error) {
	currentTargets, err := e.getTargetsFromHTML(entry)
	if err != nil {
		if os.IsNotExist(err) {
			if entry.Deleted {
				currentTargets = []string{}
			} else {
				return nil, nil, nil, fmt.Errorf("entry should exist as it is not deleted %s: %w", entry.ID, err)
			}
		} else {
			return nil, nil, nil, err
		}
	}

	entryData, err := e.GetEntryData(entry)
	if err != nil {
		return nil, nil, nil, err
	}

	oldTargets := entryData.Targets
	oldTargets = uniqString(oldTargets)

	targets := append(currentTargets, oldTargets...)
	targets = uniqString(targets)

	return targets, currentTargets, oldTargets, nil
}

func (e *Eagle) getTargetsFromHTML(entry *Entry) ([]string, error) {
	// html, err := e.getEntryHTML(entry)
	// if err != nil {
	// 	return nil, err
	// }

	// r := bytes.NewReader(html)

	// targets, err := webmention.DiscoverLinksFromReader(r, entry.Permalink, ".h-entry .e-content a")
	// if err != nil {
	// 	return nil, err
	// }

	// targets = e.filterTargets(targets)

	// if entry.Metadata.ReplyTo != nil && entry.Metadata.ReplyTo.URL != "" {
	// 	targets = append(targets, entry.Metadata.ReplyTo.URL)
	// }

	targets := []string{}

	return uniqString(targets), nil
}

func (e *Eagle) filterTargets(targets []string) []string {
	filteredTargets := []string{}
	for _, target := range targets {
		url, err := urlpkg.Parse(target)
		if err != nil {
			continue
		}

		if url.Scheme == "http" || url.Scheme == "https" {
			filteredTargets = append(filteredTargets, target)
		}
	}
	return filteredTargets
}

func (e *Eagle) sendWebmention(source, target string) error {
	endpoint, err := e.webmentionsClient.DiscoverEndpoint(target)
	if err != nil {
		return err
	}

	if isPrivate(endpoint) {
		return fmt.Errorf("webmention endpoint is a private address: %s", endpoint)
	}

	res, err := e.webmentionsClient.SendWebmention(endpoint, source, target)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, res.Body)
	defer res.Body.Close()

	return nil
}

func (e *Eagle) UpdateTargets(entry *Entry) error {
	if entry.Deleted {
		return nil
	}

	_, curr, _, err := e.GetWebmentionTargets(entry)
	if err != nil {
		return err
	}

	if len(curr) == 0 {
		return nil
	}

	return e.TransformEntryData(entry, func(data *EntryData) (*EntryData, error) {
		data.Targets = curr
		return data, nil
	})
}

func (e *Eagle) ReceiveWebmentions(payload *WebmentionPayload) error {
	e.log.Infow("received webmention", "webmention", payload)

	url, err := urlpkg.Parse(payload.Target)
	if err != nil {
		return fmt.Errorf("invalid target: %s", payload.Target)
	}

	entry, err := e.GetEntry(url.Path)
	if err != nil {
		return err
	}

	if payload.Deleted {
		return e.TransformEntryData(entry, func(data *EntryData) (*EntryData, error) {
			newWebmentions := []*Webmention{}
			for _, mention := range data.Webmentions {
				if mention.URL != payload.Source {
					newWebmentions = append(newWebmentions, mention)
				}
			}
			data.Webmentions = newWebmentions
			return data, nil
		})
	}

	newWebmention, err := e.parseWebmentionPayload(payload)
	if err != nil {
		return err
	}

	return e.TransformEntryData(entry, func(data *EntryData) (*EntryData, error) {
		for i, webmention := range data.Webmentions {
			if webmention.URL == newWebmention.URL {
				data.Webmentions[i] = newWebmention
				return data, nil
			}
		}

		data.Webmentions = append(data.Webmentions, newWebmention)
		return data, nil
	})
}

func (e *Eagle) parseWebmentionPayload(payload *WebmentionPayload) (*Webmention, error) {
	ee := &Webmention{
		XRay: XRay{
			Author: &payload.Post.Author,
		},
		WmID:    payload.Post.WmID,
		Private: payload.Post.WmPrivate,
	}

	if payload.Post.Content.Text != "" {
		ee.Content = payload.Post.Content.Text
	} else if payload.Post.Content.HTML != "" {
		ee.Content = payload.Post.Content.HTML
	}

	if ee.Content != "" {
		ee.Content = cleanContent(ee.Content)
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
