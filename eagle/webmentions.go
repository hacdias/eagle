package eagle

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	urlpkg "net/url"
	"os"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/entry/mf2"
	"github.com/hacdias/eagle/v4/xray"
	"github.com/hashicorp/go-multierror"
	"github.com/thoas/go-funk"
	"willnorris.com/go/webmention"
)

type WebmentionPayload struct {
	Source  string                 `json:"source"`
	Secret  string                 `json:"secret"`
	Deleted bool                   `json:"deleted"`
	Target  string                 `json:"target"`
	Post    map[string]interface{} `json:"post"`
}

func (e *Eagle) SendWebmentions(entry *entry.Entry) error {
	if entry.NoSendInteractions || entry.Draft {
		return nil
	}

	all, curr, _, err := e.GetWebmentionTargets(entry)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	for _, target := range all {
		// if strings.HasPrefix(target, e.Config.Site.BaseURL) {
		// TODO: it is a self-mention.
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

	sidecar, err := e.GetSidecar(entry)
	if err != nil {
		return nil, nil, nil, err
	}

	oldTargets := sidecar.Targets
	oldTargets = funk.UniqString(oldTargets)

	targets := append(currentTargets, oldTargets...)
	targets = funk.UniqString(targets)

	return targets, currentTargets, oldTargets, nil
}

func (e *Eagle) getTargetsFromHTML(entry *entry.Entry) ([]string, error) {
	var buf bytes.Buffer
	err := e.Render(&buf, &RenderData{
		Entry: entry,
	}, EntryTemplates(entry))
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

func (e *Eagle) DeleteWebmention(ee *entry.Entry, source string) error {
	return e.UpdateSidecar(ee, func(sidecar *Sidecar) (*Sidecar, error) {
		for i, reply := range sidecar.Replies {
			if reply.URL == source {
				sidecar.Replies = append(sidecar.Replies[:i], sidecar.Replies[i+1:]...)
				return sidecar, nil
			}
		}

		for i, reply := range sidecar.Interactions {
			if reply.URL == source {
				sidecar.Interactions = append(sidecar.Interactions[:i], sidecar.Interactions[i+1:]...)
				return sidecar, nil
			}
		}

		return sidecar, nil
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
		return e.DeleteWebmention(entry, payload.Source)
	}

	parsed := e.XRay.Parse(payload.Post)
	parsed.URL = payload.Source

	if parsed.Author.URL != "" {
		parsed.Author.URL = e.safeUploadFromURL("wm", parsed.Author.URL, true)
	}

	isInteraction := IsPostInteraction(parsed)

	err = e.UpdateSidecar(entry, func(sidecar *Sidecar) (*Sidecar, error) {
		var mentions []*xray.Post
		if isInteraction {
			mentions = sidecar.Interactions
		} else {
			mentions = sidecar.Replies
		}

		replaced := false
		for i, mention := range mentions {
			if mention.URL == payload.Source || mention.URL == parsed.URL {
				mentions[i] = parsed
				replaced = true
				break
			}
		}

		if !replaced {
			mentions = append(mentions, parsed)
		}

		if isInteraction {
			sidecar.Interactions = mentions
		} else {
			sidecar.Replies = mentions
		}

		return sidecar, nil
	})

	if err != nil {
		e.Notifier.Error(err)
	} else if payload.Deleted {
		e.Notifier.Info("💬 Deleted webmention at " + payload.Target)
	} else {
		e.Notifier.Info("💬 Received webmention at " + payload.Target)
	}

	e.RemoveCache(entry)
	return err
}

func isPrivate(urlStr string) bool {
	url, _ := urlpkg.Parse(urlStr)
	if url == nil {
		return false
	}

	hostname := url.Hostname()
	if hostname == "localhost" {
		return true
	}

	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}

	return ip.IsPrivate() || ip.IsLoopback()
}

func IsPostInteraction(post *xray.Post) bool {
	return post.Type == mf2.TypeLike ||
		post.Type == mf2.TypeRepost ||
		post.Type == mf2.TypeBookmark ||
		post.Type == mf2.TypeRsvp
}
