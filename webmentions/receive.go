package webmentions

import (
	"fmt"
	urlpkg "net/url"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/xray"
	"github.com/thoas/go-funk"
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

	return ws.AddOrUpdateWebmention(url.Path, parsed, payload.Source)
}

func (ws *Webmentions) AddOrUpdateWebmention(id string, post *xray.Post, sources ...string) error {
	e, err := ws.fs.GetEntry(id)
	if err != nil {
		return err
	}

	isInteraction := IsInteraction(post)

	err = ws.fs.UpdateSidecar(e, func(sidecar *eagle.Sidecar) (*eagle.Sidecar, error) {
		var mentions []*eagle.Mention
		if isInteraction {
			mentions = sidecar.Interactions
		} else {
			mentions = sidecar.Replies
		}

		replaced := false
		for i, mention := range mentions {
			if funk.ContainsString(sources, mention.URL) || mention.URL == post.URL {
				mentions[i] = &eagle.Mention{Post: *post, Hidden: mentions[i].Hidden}
				replaced = true
				break
			}
		}

		if !replaced {
			mentions = append(mentions, &eagle.Mention{Post: *post})
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
		ws.notifier.Info("💬 Received webmention at " + e.Permalink)
	}

	return err
}

func (ws *Webmentions) DeleteWebmention(id, source string) error {
	e, err := ws.fs.GetEntry(id)
	if err != nil {
		return err
	}

	err = ws.fs.UpdateSidecar(e, func(sidecar *eagle.Sidecar) (*eagle.Sidecar, error) {
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

	if err != nil {
		ws.notifier.Error(err)
	} else {
		ws.notifier.Info("💬 Deleted webmention at " + e.Permalink)
	}

	return err
}
