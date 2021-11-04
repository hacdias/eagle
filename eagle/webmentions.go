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

	"github.com/hashicorp/go-multierror"
	"github.com/thoas/go-funk"
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

type WebmentionPayload struct {
	Source  string                 `json:"source"`
	Secret  string                 `json:"secret"`
	Deleted bool                   `json:"deleted"`
	Target  string                 `json:"target"`
	Post    map[string]interface{} `json:"post"`
}

func (e *Eagle) SendWebmentions(entry *Entry) error {
	all, curr, _, err := e.GetWebmentionTargets(entry)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	for _, target := range all {
		if strings.HasPrefix(target, e.Config.Site.BaseURL) {
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
	oldTargets = funk.UniqString(oldTargets)

	targets := append(currentTargets, oldTargets...)
	targets = funk.UniqString(targets)

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

	return funk.UniqString(targets), nil
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
	// TODO: just save as xray and add line to wms

	// url, err := urlpkg.Parse(payload.Target)
	// if err != nil {
	// 	return fmt.Errorf("invalid target: %s", payload.Target)
	// }

	// entry, err := e.GetEntry(url.Path)
	// if err != nil {
	// 	return err
	// }

	// if payload.Deleted {
	// 	return e.TransformEntryData(entry, func(data *EntryData) (*EntryData, error) {
	// 		newWebmentions := []*Webmention{}
	// 		for _, mention := range data.Webmentions {
	// 			if mention.URL != payload.Source {
	// 				newWebmentions = append(newWebmentions, mention)
	// 			}
	// 		}
	// 		data.Webmentions = newWebmentions
	// 		return data, nil
	// 	})
	// }

	// newWebmention, err := e.parseWebmentionPayload(payload)
	// if err != nil {
	// 	return err
	// }

	// return e.TransformEntryData(entry, func(data *EntryData) (*EntryData, error) {
	// 	for i, webmention := range data.Webmentions {
	// 		if webmention.URL == newWebmention.URL {
	// 			data.Webmentions[i] = newWebmention
	// 			return data, nil
	// 		}
	// 	}

	// 	data.Webmentions = append(data.Webmentions, newWebmention)
	// 	return data, nil
	// })

	return nil
}

func (e *Eagle) uploadXRayAuthorPhoto(url string) string {
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
