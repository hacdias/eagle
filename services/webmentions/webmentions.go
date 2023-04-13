package webmentions

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	urlpkg "net/url"
	"time"

	"github.com/hacdias/eagle/core"
	"github.com/hacdias/eagle/log"
	"github.com/hashicorp/go-multierror"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"willnorris.com/go/webmention"
)

type Webmentions struct {
	log      *zap.SugaredLogger
	client   *webmention.Client
	fs       *core.FS
	hugo     *core.Hugo
	notifier core.Notifier
}

func NewWebmentions(fs *core.FS, hugo *core.Hugo, notifier core.Notifier) *Webmentions {
	return &Webmentions{
		log: log.S().Named("webmentions"),
		client: webmention.New(&http.Client{
			Timeout: time.Minute,
		}),
		fs:       fs,
		hugo:     hugo,
		notifier: notifier,
	}
}

func (ws *Webmentions) EntryHook(old, new *core.Entry) error {
	return ws.SendWebmentions(old, new)
}

type Webmention struct {
	Source  string                 `json:"source"`
	Secret  string                 `json:"secret"`
	Deleted bool                   `json:"deleted"`
	Target  string                 `json:"target"`
	Post    map[string]interface{} `json:"post"`
}

func (ws *Webmentions) ReceiveWebmentions(payload *Webmention) error {
	ws.log.Infow("received webmention", "webmention", payload)

	target, err := urlpkg.Parse(payload.Target)
	if err != nil {
		return fmt.Errorf("invalid target: %s", payload.Target)
	}

	source, err := urlpkg.Parse(payload.Source)
	if err != nil {
		return fmt.Errorf("invalid source: %s", payload.Source)
	}

	if payload.Deleted || source.Hostname() == target.Hostname() {
		// Deletions and self-webmentions are ignored.
		return nil
	}

	// Make sure entry actually exists to avoid useless notifications.
	e, err := ws.fs.GetEntry(target.Path)
	if err != nil {
		return err
	}

	ws.notifier.Info(fmt.Sprintf("💬 #webmention on %s, via %s.", e.Permalink, payload.Source))
	return nil
}

func (ws *Webmentions) SendWebmentions(old, new *core.Entry) error {
	var targets []string

	if canSendWebmentions(old) {
		oldTargets, err := ws.getTargetsFromHTML(old)
		if err != nil {
			return err
		}
		targets = append(targets, oldTargets...)
	}

	if canSendWebmentions(new) {
		newTargets, err := ws.getTargetsFromHTML(new)
		if err != nil {
			return err
		}
		targets = append(targets, newTargets...)
	}

	targets = lo.Uniq(targets)

	var errs *multierror.Error

	for _, target := range targets {
		err := ws.sendWebmention(new.Permalink, target)
		if err != nil && !errors.Is(err, webmention.ErrNoEndpointFound) {
			err = fmt.Errorf("send webmention error %s: %w", target, err)
			errs = multierror.Append(errs, err)
		}
	}

	err := errs.ErrorOrNil()
	if err != nil {
		return fmt.Errorf("webmention errors for %s: %w", new.ID, err)
	}

	return nil
}

func (ws *Webmentions) getTargetsFromHTML(entry *core.Entry) ([]string, error) {
	html, err := ws.hugo.GetEntryHTML(entry)
	if err != nil {
		return nil, err
	}

	targets, err := webmention.DiscoverLinksFromReader(bytes.NewReader(html), entry.Permalink, ".h-entry .e-content a, .h-entry .h-cite a")
	if err != nil {
		return nil, err
	}

	targets = (lo.Filter(targets, func(target string, _ int) bool {
		url, err := urlpkg.Parse(target)
		if err != nil {
			return false
		}

		return url.Scheme == "http" || url.Scheme == "https"
	}))

	return lo.Uniq(targets), nil
}

func (ws *Webmentions) sendWebmention(source, target string) error {
	endpoint, err := ws.client.DiscoverEndpoint(target)
	if err != nil {
		return fmt.Errorf("err discovering endpoint: %w", err)
	}

	if isPrivate(endpoint) {
		return fmt.Errorf("webmention endpoint is a private address: %s", endpoint)
	}

	res, err := ws.client.SendWebmention(endpoint, source, target)
	if err != nil {
		return fmt.Errorf("err sending webmention: %w", err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	defer res.Body.Close()

	return nil
}

func canSendWebmentions(e *core.Entry) bool {
	return e != nil &&
		!e.NoSendInteractions &&
		!e.Draft &&
		!e.Deleted() &&
		!e.IsList()
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
