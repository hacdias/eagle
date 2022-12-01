package webmentions

import (
	"fmt"
	urlpkg "net/url"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/xray"
	"github.com/samber/lo"
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

	url, err := urlpkg.Parse(payload.Target)
	if err != nil {
		return fmt.Errorf("invalid target: %s", payload.Target)
	}

	if payload.Deleted {
		return ws.DeleteWebmention(url.Path, payload.Source)
	}

	parsed := xray.Parse(payload.Post)
	parsed.URL = payload.Source

	if parsed.Author.Photo != "" {
		parsed.Author.Photo = ws.media.SafeUploadFromURL("wm", parsed.Author.Photo, true)
	}

	return ws.AddOrUpdateWebmention(url.Path, &eagle.Mention{Post: *parsed}, payload.Source)
}

func (ws *Webmentions) AddOrUpdateWebmention(id string, mention *eagle.Mention, sourcesOrIDs ...string) error {
	e, err := ws.fs.GetEntry(id)
	if err != nil {
		return err
	}

	isInteraction := isInteraction(mention)

	updated := false
	err = ws.fs.UpdateSidecar(e, func(sidecar *eagle.Sidecar) (*eagle.Sidecar, error) {
		var mentions []*eagle.Mention
		if isInteraction {
			mentions = sidecar.Interactions
		} else {
			mentions = sidecar.Replies
		}

		for i, m := range mentions {
			if (m.URL == mention.URL && len(m.URL) != 0) || (m.ID == mention.ID && len(m.ID) != 0) ||
				lo.Contains(sourcesOrIDs, m.URL) || lo.Contains(sourcesOrIDs, m.ID) {
				mention.Hidden = mentions[i].Hidden
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

	if err != nil {
		ws.notifier.Error(err)
	} else {
		action := "received"
		if updated {
			action += "updated"
		}
		ws.notifier.Info(fmt.Sprintf("💬 #webmention %s on %s, via %s.", action, e.Permalink, mention.URL))
	}

	return err
}

func (ws *Webmentions) DeleteWebmention(id, urlOrID string) error {
	e, err := ws.fs.GetEntry(id)
	if err != nil {
		return err
	}

	err = ws.fs.UpdateSidecar(e, func(sidecar *eagle.Sidecar) (*eagle.Sidecar, error) {
		for i, reply := range sidecar.Replies {
			if reply.URL == urlOrID || reply.ID == urlOrID {
				sidecar.Replies = append(sidecar.Replies[:i], sidecar.Replies[i+1:]...)
				return sidecar, nil
			}
		}

		for i, reply := range sidecar.Interactions {
			if reply.URL == urlOrID || reply.ID == urlOrID {
				sidecar.Interactions = append(sidecar.Interactions[:i], sidecar.Interactions[i+1:]...)
				return sidecar, nil
			}
		}

		return sidecar, nil
	})

	if err != nil {
		ws.notifier.Error(err)
	} else {
		ws.notifier.Info(fmt.Sprintf("💬 #webmention deleted on %s, via %s.", e.Permalink, urlOrID))
	}

	return err
}
