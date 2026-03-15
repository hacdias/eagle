package webarchive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

var (
	_ server.HookPlugin  = &WebArchive{}
	_ server.QueuePlugin = &WebArchive{}
)

const queueItemType = "webarchive"

func init() {
	server.RegisterPlugin("webarchive", NewWebArchive)
}

type WebArchive struct {
	core          *core.Core
	archiveOnSave bool
}

func NewWebArchive(co *core.Core, config map[string]any) (server.Plugin, error) {
	cfg := typed.New(config)

	return &WebArchive{
		core:          co,
		archiveOnSave: cfg.Bool("archiveonsave"),
	}, nil
}

func (wa *WebArchive) PreSaveHook(*core.Entry) error {
	return nil
}

func (wa *WebArchive) PostSaveHook(e *core.Entry, isNew bool) error {
	if !wa.archiveOnSave || !isNew || e.Deleted() {
		return nil
	}

	links, err := wa.core.GetEntryLinks(e, false)
	if err != nil {
		return fmt.Errorf("failed to get entry links: %w", err)
	}

	for _, link := range links {
		_ = wa.core.Enqueue(context.Background(), queueItemType, queueItemPayload{
			URL: link,
		})
	}

	return nil
}

func (wa *WebArchive) QueueItemType() string {
	return queueItemType
}

type queueItemPayload struct {
	URL string
}

func (wa *WebArchive) HandleQueueItem(ctx context.Context, payload []byte) error {
	var p queueItemPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}

	_, err := webArchive(p.URL)
	return err
}

func webArchive(url string) (string, error) {
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head("https://web.archive.org/save/" + url)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status code not ok: %d", resp.StatusCode)
	}

	location, err := resp.Location()
	if err != nil {
		return "", err
	}

	return location.String(), nil
}
