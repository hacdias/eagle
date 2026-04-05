package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"go.hacdias.com/eagle/core"
	"willnorris.com/go/webmention"
)

const (
	webmentionPath      = "/webmention"
	commentsPath        = "/comments"
	queueTypeWebmention = "webmention"

	webmentionHTTPTimeout = 30 * time.Second
)

func (s *Server) commentsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("parse form failed: %w", err))
		return
	}

	// Anti-spam prevention with user-defined captcha value.
	if s.c.Comments.Captcha != "" && s.c.Comments.Captcha != strings.ToLower(r.Form.Get("captcha")) {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("anti-spam verification failed"))
		return
	}

	name := r.Form.Get("name")
	website := r.Form.Get("website")
	content := r.Form.Get("content")
	target := r.Form.Get("target")

	if target == "" {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("target entry is missing"))
		return
	}

	e, err := s.core.GetEntry(target)
	if err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("target entry is invalid: %w", err))
		return
	}

	// Sanitize things just in case, specially the content.
	sanitize := bluemonday.StrictPolicy()
	name = sanitize.Sanitize(name)
	website = sanitize.Sanitize(website)
	content = sanitize.Sanitize(content)

	if len(name) == 0 || len(content) == 0 {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("name and content are required"))
		return
	}

	if len(content) > 1000 || len(name) > 200 || len(website) > 200 {
		s.serveErrorHTML(w, r, http.StatusBadRequest, errors.New("content, name, or website outside of limits"))
		return
	}

	if _, err := url.Parse(website); err != nil {
		s.serveErrorHTML(w, r, http.StatusBadRequest, fmt.Errorf("website url is invalid: %w", err))
		return
	}

	s.log.Infow("received comment entry", "name", name, "website", website, "content", content)

	err = s.core.DB().CreateMention(r.Context(), &core.Mention{
		ID: uuid.New().String(),
		XRay: core.XRay{
			Author:    name,
			AuthorURL: website,
			Content:   content,
			Date:      time.Now(),
		},
		EntryID: e.ID,
	})
	if err != nil {
		s.panelError(w, r, http.StatusInternalServerError, err)
		return
	}

	s.n.Notify(fmt.Sprintf("💬 #mention pending approval for %q", e.Permalink))
	http.Redirect(w, r, s.c.Comments.Redirect, http.StatusSeeOther)
}

type wmPayload struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// See https://www.w3.org/TR/webmention/#receiving-webmentions
func (s *Server) webmentionPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	sourceStr := r.FormValue("source")
	targetStr := r.FormValue("target")

	if sourceStr == "" || targetStr == "" {
		http.Error(w, "source and target are required", http.StatusBadRequest)
		return
	}

	sourceURL, err := url.Parse(sourceStr)
	if err != nil || (sourceURL.Scheme != "http" && sourceURL.Scheme != "https") {
		http.Error(w, "source must be a valid http or https URL", http.StatusBadRequest)
		return
	}

	targetURL, err := url.Parse(targetStr)
	if err != nil || (targetURL.Scheme != "http" && targetURL.Scheme != "https") {
		http.Error(w, "target must be a valid http or https URL", http.StatusBadRequest)
		return
	}

	// Source and target must differ.
	if sourceStr == targetStr {
		http.Error(w, "source and target must be different", http.StatusBadRequest)
		return
	}

	// Target must be on our domain.
	baseURL, _ := url.Parse(s.c.Site.BaseURL)
	if targetURL.Hostname() != baseURL.Hostname() {
		http.Error(w, "target is not on this site", http.StatusBadRequest)
		return
	}

	// Target must resolve to a known entry.
	_, err = s.core.GetEntryByPermalink(targetStr)
	if err != nil {
		http.Error(w, "target not found", http.StatusBadRequest)
		return
	}

	if err := s.core.Enqueue(r.Context(), queueTypeWebmention, wmPayload{
		Source: sourceStr,
		Target: targetStr,
	}); err != nil {
		s.log.Errorw("failed to enqueue webmention", "source", sourceStr, "target", targetStr, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	s.log.Infow("webmention enqueued", "source", sourceStr, "target", targetStr)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleWebmentionQueueItem(ctx context.Context, payload []byte) error {
	var job wmPayload
	if err := json.Unmarshal(payload, &job); err != nil {
		return &core.PermanentError{Err: fmt.Errorf("invalid webmention payload: %w", err)}
	}

	s.log.Infow("processing webmention", "source", job.Source, "target", job.Target)

	e, err := s.core.GetEntryByPermalink(job.Target)
	if err != nil {
		return &core.PermanentError{Err: fmt.Errorf("target entry not found for %s: %w", job.Target, err)}
	}

	// Reject private/loopback source URLs to prevent SSRF.
	if core.IsPrivateURL(job.Source) {
		return &core.PermanentError{Err: fmt.Errorf("source %s is a private address", job.Source)}
	}

	// Fetch the source page.
	httpClient := &http.Client{
		Timeout: webmentionHTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 20 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.Source, nil)
	if err != nil {
		return fmt.Errorf("creating request for source %s: %w", job.Source, err)
	}
	req.Header.Set("Accept", "text/html, application/xhtml+xml")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching source %s: %w", job.Source, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		// Source has been deleted — remove any existing mention.
		if delErr := s.core.DeleteWebmention(e.ID, job.Source); delErr != nil {
			s.log.Errorw("failed to delete webmention for gone source", "source", job.Source, "err", delErr)
		} else {
			s.n.Notify(fmt.Sprintf("💬 #mention deleted for %q: %q", e.Permalink, job.Source))
		}
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("source %s returned status %d", job.Source, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5 MB cap
	if err != nil {
		return fmt.Errorf("reading source body %s: %w", job.Source, err)
	}

	links, err := webmention.DiscoverLinksFromReader(bytes.NewReader(body), job.Source, "")
	if err != nil {
		return fmt.Errorf("parsing source HTML %s: %w", job.Source, err)
	}

	target := strings.TrimSuffix(job.Target, "/")
	contains := false
	for _, href := range links {
		if href == target || strings.HasPrefix(href, target+"/") || strings.HasPrefix(href, target+"?") || strings.HasPrefix(href, target+"#") {
			contains = true
			break
		}
	}

	// Verify source contains a link to target.
	if !contains {
		return &core.PermanentError{Err: fmt.Errorf("source %s does not link to target %s", job.Source, job.Target)}
	}

	// Parse microformats from source.
	sourceURL, _ := url.Parse(job.Source)
	post := core.ParseXRay(bytes.NewReader(body), sourceURL)

	// Upsert: update existing mention if one exists for this source+entry.
	existing, err := s.core.DB().GetMentionBySourceAndEntry(ctx, job.Source, e.ID)
	if err == nil {
		existing.XRay = *post
		if err := s.core.DB().UpdateMention(ctx, existing); err != nil {
			return fmt.Errorf("updating webmention: %w", err)
		}
		s.n.Notify(fmt.Sprintf("💬 #mention updated for %q: %q", e.Permalink, job.Source))
		return nil
	}

	mention := &core.Mention{
		ID:      uuid.New().String(),
		XRay:    *post,
		EntryID: e.ID,
		Source:  job.Source,
	}

	if err := s.core.DB().CreateMention(ctx, mention); err != nil {
		return fmt.Errorf("storing webmention: %w", err)
	}

	s.n.Notify(fmt.Sprintf("💬 #mention pending approval for %q: %q", e.Permalink, job.Source))
	return nil
}
