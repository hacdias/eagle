package webmentions

import (
	"fmt"
	urlpkg "net/url"
)

type Payload struct {
	Source  string                 `json:"source"`
	Secret  string                 `json:"secret"`
	Deleted bool                   `json:"deleted"`
	Target  string                 `json:"target"`
	Post    map[string]interface{} `json:"post"`
}

func (ws *Webmentions) ReceiveWebmentions(payload *Payload) error {
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
