package webmentions

import (
	"fmt"
	urlpkg "net/url"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/xray"
)

func (ws *WebmentionsService) ReceiveWebmentions(payload *WebmentionPayload) error {
	// wip: e.log.Infow("received webmention", "webmention", payload)

	url, err := urlpkg.Parse(payload.Target)
	if err != nil {
		return fmt.Errorf("invalid target: %s", payload.Target)
	}

	entry, err := ws.Eagle.GetEntry(url.Path)
	if err != nil {
		return err
	}

	if payload.Deleted {
		return ws.DeleteWebmention(entry, payload.Source)
	}

	parsed := xray.Parse(payload.Post)
	parsed.URL = payload.Source

	if parsed.Author.Photo != "" {
		parsed.Author.Photo = ws.Eagle.SafeUploadFromURL("wm", parsed.Author.Photo, true)
	}

	isInteraction := IsInteraction(parsed)

	err = ws.Eagle.UpdateSidecar(entry, func(sidecar *eagle.Sidecar) (*eagle.Sidecar, error) {
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
		ws.Eagle.Notifier.Error(err)
	} else if payload.Deleted {
		ws.Eagle.Notifier.Info("💬 Deleted webmention at " + payload.Target)
	} else {
		ws.Eagle.Notifier.Info("💬 Received webmention at " + payload.Target)
	}

	ws.Eagle.RemoveCache(entry)
	return err
}

func (ws *WebmentionsService) DeleteWebmention(ee *entry.Entry, source string) error {
	return ws.Eagle.UpdateSidecar(ee, func(sidecar *eagle.Sidecar) (*eagle.Sidecar, error) {
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
