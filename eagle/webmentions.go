package eagle

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path"

	"github.com/hacdias/eagle/v2/entry"
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

func (e *Eagle) SendWebmentions(entry *entry.Entry) error {
	if entry.NoSendInteractions {
		return nil
	}

	all, curr, _, err := e.GetWebmentionTargets(entry)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	for _, target := range all {
		// if strings.HasPrefix(target, e.Config.Site.BaseURL) {
		// TODO(future): it is a self-mention.
		// }

		err := e.sendWebmention(entry.Permalink, target)
		if err != nil && !errors.Is(err, webmention.ErrNoEndpointFound) {
			err = fmt.Errorf("send webmention error %s: %w", target, err)
			errs = multierror.Append(errs, err)
		}
	}

	if !entry.Deleted {
		// If it's not a deleted entry, update the targets list.
		err = e.UpdateSidecar(entry, func(data *Sidecar) (*Sidecar, error) {
			data.Targets = curr
			return data, nil
		})

		errs = multierror.Append(errs, err)
	}

	err = errs.ErrorOrNil()
	if err == nil {
		return nil
	}

	return fmt.Errorf("webmention errors for %s: %w", entry.ID, err)
}

func (e *Eagle) GetWebmentionTargets(entry *entry.Entry) ([]string, []string, []string, error) {
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

	entryData, err := e.GetSidecar(entry)
	if err != nil {
		return nil, nil, nil, err
	}

	oldTargets := entryData.Targets
	oldTargets = funk.UniqString(oldTargets)

	targets := append(currentTargets, oldTargets...)
	targets = funk.UniqString(targets)

	return targets, currentTargets, oldTargets, nil
}

func (e *Eagle) getTargetsFromHTML(entry *entry.Entry) ([]string, error) {
	var buf bytes.Buffer
	err := e.Render(&buf, &RenderData{
		Entry: entry,
	}, e.EntryTemplates(entry))
	if err != nil {
		return nil, err
	}

	targets, err := webmention.DiscoverLinksFromReader(&buf, entry.Permalink, ".h-entry .e-content a, .h-entry .h-cite a")
	if err != nil {
		return nil, err
	}

	targets = (funk.FilterString(targets, func(target string) bool {
		url, err := urlpkg.Parse(target)
		if err != nil {
			return false
		}

		return url.Scheme == "http" || url.Scheme == "https"
	}))

	return funk.UniqString(targets), nil
}

func (e *Eagle) sendWebmention(source, target string) error {
	endpoint, err := e.wmClient.DiscoverEndpoint(target)
	if err != nil {
		return err
	}

	if isPrivate(endpoint) {
		return fmt.Errorf("webmention endpoint is a private address: %s", endpoint)
	}

	res, err := e.wmClient.SendWebmention(endpoint, source, target)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, res.Body)
	defer res.Body.Close()

	return nil
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
		return e.UpdateSidecar(entry, func(sidecar *Sidecar) (*Sidecar, error) {
			for i, mention := range sidecar.Webmentions {
				url, ok := mention["url"].(string)
				if !ok {
					continue
				}
				if url == payload.Source {
					sidecar.Webmentions = append(sidecar.Webmentions[:i], sidecar.Webmentions[i+1:]...)
					break
				}
			}
			return sidecar, nil
		})
	}

	data := e.parseXRay(payload.Post)
	return e.UpdateSidecar(entry, func(sidecar *Sidecar) (*Sidecar, error) {
		for i, mention := range sidecar.Webmentions {
			url, ok := mention["url"].(string)
			if !ok {
				continue
			}

			if url == payload.Source {
				sidecar.Webmentions[i] = data
				return sidecar, nil
			}
		}

		sidecar.Webmentions = append(sidecar.Webmentions, data)
		return sidecar, nil
	})

	// TODO(vw)
	// go func() {
	// 	// INVALIDATE CACHE OR STH
	// 	if wm.Deleted {
	// 		s.Notify("💬 Deleted webmention at " + wm.Target)
	// 	} else {
	// 		s.Notify("💬 Received webmention at " + wm.Target)
	// 	}
	// }()
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
