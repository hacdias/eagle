package core

import (
	"errors"
	"fmt"
	"io"
	"net"
	urlpkg "net/url"

	"github.com/samber/lo"
	"willnorris.com/go/webmention"
)

func (co *Core) AddOrUpdateWebmention(id string, mention *Mention, sourceOrURL string) error {
	e, err := co.GetEntry(id)
	if err != nil {
		return err
	}

	isInteraction := mention.IsInteraction()

	return co.UpdateSidecar(e, func(sidecar *Sidecar) (*Sidecar, error) {
		var mentions []*Mention
		if isInteraction {
			mentions = sidecar.Interactions
		} else {
			mentions = sidecar.Replies
		}

		updated := false
		for i, m := range mentions {
			if (m.URL == mention.URL && len(m.URL) != 0) ||
				(m.Source == mention.Source && len(m.Source) != 0) ||
				(m.URL == sourceOrURL && len(m.URL) != 0) ||
				(m.Source == sourceOrURL && len(m.Source) != 0) {
				mentions[i] = mention
				updated = true
				break
			}
		}

		if !updated {
			mentions = append(mentions, mention)
		}

		if isInteraction {
			sidecar.Interactions = mentions
		} else {
			sidecar.Replies = mentions
		}

		return sidecar, nil
	})
}

func (co *Core) DeleteWebmention(id, sourceOrURL string) error {
	e, err := co.GetEntry(id)
	if err != nil {
		return err
	}

	return co.UpdateSidecar(e, func(sidecar *Sidecar) (*Sidecar, error) {
		sidecar.Replies = lo.Filter(sidecar.Replies, func(mention *Mention, _ int) bool {
			return mention.URL != sourceOrURL && mention.Source != sourceOrURL
		})

		sidecar.Interactions = lo.Filter(sidecar.Interactions, func(mention *Mention, _ int) bool {
			return mention.URL != sourceOrURL && mention.Source != sourceOrURL
		})

		return sidecar, nil
	})
}

func (co *Core) SendWebmentions(e *Entry) error {
	targets, err := co.GetEntryLinks(e.Permalink)
	if err != nil {
		return err
	}
	targets = lo.Uniq(targets)

	for _, target := range targets {
		wmErr := co.sendWebmention(e.Permalink, target)
		if wmErr != nil && !errors.Is(wmErr, webmention.ErrNoEndpointFound) {
			wmErr = fmt.Errorf("send webmention error %s: %w", target, wmErr)
			err = errors.Join(err, wmErr)
		}
	}

	return err
}

func (co *Core) sendWebmention(source, target string) error {
	endpoint, err := co.wmClient.DiscoverEndpoint(target)
	if err != nil {
		return fmt.Errorf("error discovering endpoint: %w", err)
	}

	if isPrivate(endpoint) {
		return fmt.Errorf("webmention endpoint is a private address: %s", endpoint)
	}

	res, err := co.wmClient.SendWebmention(endpoint, source, target)
	if err != nil {
		return fmt.Errorf("erro sending webmention: %w", err)
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
