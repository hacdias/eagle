package core

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	urlpkg "net/url"
	"os"
	"syscall"
	"time"

	"github.com/samber/lo"
	"willnorris.com/go/webmention"
)

func (co *Core) AddOrUpdateWebmention(id string, mention *Mention, sourceOrURL string) error {
	e, err := co.GetEntry(id)
	if err != nil {
		return err
	}

	source, err := urlpkg.Parse(mention.URL)
	if err == nil && source.Hostname() == co.baseURL.Hostname() {
		// Self-webmentions are ignored.
		return nil
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
	if sourceOrURL == "" {
		return nil
	}

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

func (co *Core) SendWebmentions(e *Entry, otherTargets ...string) error {
	if !e.IsPost() || e.Draft || e.Future() {
		return nil
	}

	targets, err := co.GetEntryLinks(e, true)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	targets = append(targets, otherTargets...)
	targets = lo.Uniq(targets)

	err = nil
	for _, target := range targets {
		targetUrl, targetErr := urlpkg.Parse(target)
		if targetErr == nil && targetUrl.Hostname() == co.baseURL.Hostname() {
			// Self-webmentions are ignored.
			continue
		}

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

	res, err := co.wmClient.SendWebmention(endpoint, source, target)
	if err != nil {
		return fmt.Errorf("error sending webmention: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	return nil
}

// PublicTransport returns an HTTP transport that only connects to public
// addresses. The check happens when connecting, such that it also applies to
// hostnames resolving to non-public addresses and to redirects.
func PublicTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   checkPublicAddress,
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext
	return transport
}

func checkPublicAddress(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}

	ip := net.ParseIP(host)
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("refusing to connect to non-public address %s", host)
	}

	return nil
}
