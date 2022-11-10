package webmentions

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	urlpkg "net/url"
	"os"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/renderer"
	"github.com/hashicorp/go-multierror"
	"github.com/thoas/go-funk"
	"willnorris.com/go/webmention"
)

func (ws *Webmentions) SendWebmentions(e *eagle.Entry) error {
	if e.NoSendInteractions ||
		e.Draft {
		return nil
	}

	all, curr, _, err := ws.GetWebmentionTargets(e)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	for _, target := range all {
		// if strings.HasPrefix(target, e.Config.Site.BaseURL) {
		// TODO: it is a self-mention.
		// }

		err := ws.sendWebmention(e.Permalink, target)
		if err != nil && !errors.Is(err, webmention.ErrNoEndpointFound) {
			err = fmt.Errorf("send webmention error %s: %w", target, err)
			errs = multierror.Append(errs, err)
		}
	}

	if !e.Deleted {
		// If it's not a deleted entry, update the targets list.
		err = ws.fs.UpdateSidecar(e, func(data *eagle.Sidecar) (*eagle.Sidecar, error) {
			data.Targets = curr
			return data, nil
		})

		errs = multierror.Append(errs, err)
	}

	err = errs.ErrorOrNil()
	if err == nil {
		return nil
	}

	return fmt.Errorf("webmention errors for %s: %w", e.ID, err)
}

func (ws *Webmentions) GetWebmentionTargets(entry *eagle.Entry) ([]string, []string, []string, error) {
	currentTargets, err := ws.getTargetsFromHTML(entry)
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

	sidecar, err := ws.fs.GetSidecar(entry)
	if err != nil {
		return nil, nil, nil, err
	}

	oldTargets := sidecar.Targets
	oldTargets = funk.UniqString(oldTargets)

	targets := append(currentTargets, oldTargets...)
	targets = funk.UniqString(targets)

	return targets, currentTargets, oldTargets, nil
}

func (ws *Webmentions) getTargetsFromHTML(entry *eagle.Entry) ([]string, error) {
	var buf bytes.Buffer
	err := ws.renderer.Render(&buf, &renderer.RenderData{
		Entry: entry,
	}, renderer.EntryTemplates(entry))
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

func (ws *Webmentions) sendWebmention(source, target string) error {
	endpoint, err := ws.client.DiscoverEndpoint(target)
	if err != nil {
		return err
	}

	if isPrivate(endpoint) {
		return fmt.Errorf("webmention endpoint is a private address: %s", endpoint)
	}

	res, err := ws.client.SendWebmention(endpoint, source, target)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, res.Body)
	defer res.Body.Close()

	return nil
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
