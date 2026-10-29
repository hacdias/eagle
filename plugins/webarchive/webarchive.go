package webarchive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

var (
	_ server.PostSaveHookPlugin = &WebArchive{}
	_ server.QueuePlugin        = &WebArchive{}
)

const queueItemType = "webarchive"

func init() {
	server.RegisterPlugin("webarchive", NewWebArchive)
}

type WebArchive struct {
	core       *core.Core
	httpClient *http.Client
	userAgent  string
	enabled    bool
	accessKey  string
	secretKey  string
}

type webArchiveConfig struct {
	Enabled   bool
	AccessKey string
	SecretKey string
}

func NewWebArchive(co *core.Core, config webArchiveConfig) (server.Plugin, error) {
	if config.AccessKey == "" {
		return nil, errors.New("webarchive access key missing")
	}

	if config.SecretKey == "" {
		return nil, errors.New("webarchive secret key missing")
	}

	return &WebArchive{
		core: co,
		httpClient: &http.Client{
			Timeout: time.Minute,
		},
		userAgent: fmt.Sprintf("eagle/%s", co.BaseURL().String()),
		enabled:   config.Enabled,
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
	}, nil
}

func (wa *WebArchive) PostSaveHook(e *core.Entry, isNew bool) error {
	if !wa.enabled || !isNew || e.Deleted() {
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

	return wa.save(ctx, p.URL)
}

type saveResponse struct {
	JobID     string `json:"job_id"`
	StatusExt string `json:"status_ext"`
	Message   string `json:"message"`
}

// save submits a capture request to the Save Page Now 2 API. The capture
// itself happens asynchronously on the Wayback Machine.
func (wa *WebArchive) save(ctx context.Context, link string) error {
	body := url.Values{"url": {link}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://web.archive.org/save", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "LOW "+wa.accessKey+":"+wa.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", wa.userAgent)

	res, err := wa.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	var data saveResponse
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return fmt.Errorf("failed to decode response (status %d): %w", res.StatusCode, err)
	}

	if data.JobID == "" {
		return fmt.Errorf("save failed (status %d): %s %s", res.StatusCode, data.StatusExt, data.Message)
	}

	return nil
}
