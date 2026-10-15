package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"sort"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/karlseguin/typed"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"gorm.io/gorm"
)

func (s *Server) getEntrySyndicationContext(e *core.Entry) (*SyndicationContext, error) {
	ctx := &SyndicationContext{}

	thumbnailStr := typed.New(e.Other).String("thumbnail")

	// Get the first 4 photos from the entry
	for _, p := range e.Photos {
		data, mimetype, err := s.media.GetImage(p.URL)
		if err != nil {
			return nil, err
		}

		photo := &Photo{
			Data:     data,
			MimeType: mimetype,
			Title:    p.Title,
			Alt:      p.Alt,
			Width:    p.Width,
			Height:   p.Height,
		}

		if p.URL == thumbnailStr {
			ctx.Thumbnail = photo
		}

		ctx.Photos = append(ctx.Photos, photo)
	}

	if ctx.Thumbnail == nil && thumbnailStr != "" {
		var err error
		data, mimetype, err := s.media.GetImage(thumbnailStr)
		if err != nil {
			return nil, fmt.Errorf("failed to get thumbnail: %w", err)
		}

		ctx.Thumbnail = &Photo{
			Data:     data,
			MimeType: mimetype,
		}

		config, _, err := image.DecodeConfig(bytes.NewReader(ctx.Thumbnail.Data))
		if err == nil {
			ctx.Thumbnail.Width = config.Width
			ctx.Thumbnail.Height = config.Height
		}
	} else if len(ctx.Photos) > 0 {
		ctx.Thumbnail = ctx.Photos[0]
	}

	return ctx, nil
}

func (s *Server) syndicate(e *core.Entry, syndicators []string, status string) error {
	if !e.IsPost() {
		return nil
	}

	// Get the syndication context
	syndicationContext, err := s.getEntrySyndicationContext(e)
	if err != nil {
		return fmt.Errorf("failed to get syndication context: %w", err)
	}
	syndicationContext.Status = status

	// Include syndicators that have already been used for this post
	for name, syndicator := range s.syndicators {
		if syndicator.IsSyndicated(e) {
			syndicators = append(syndicators, name)
		}
	}

	var errs error

	syndicators = lo.Uniq(syndicators)
	s.log.Infow("syndicating entry", "id", e.ID, "syndicators", syndicators)

	// Do the actual syndication
	for _, name := range syndicators {
		if syndicator, ok := s.syndicators[name]; ok {
			err := syndicator.Syndicate(context.Background(), e, syndicationContext)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to syndicate %s to %s: %w", e.ID, name, err))
				continue
			}
		}
	}

	// Ensure uniqueness and that it always yields the same result
	e.Syndications = lo.Uniq(e.Syndications)
	sort.Strings(e.Syndications)

	err = s.core.SaveEntry(e)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to save entry after syndication: %w", err))
	}
	s.log.Infow("syndicated entry", "id", e.ID)
	return errs
}

func (s *Server) saveEntryWithHooks(e *core.Entry, options postSaveEntryOptions) error {
	err := s.preSaveEntry(e)
	if err != nil {
		return err
	}

	err = s.core.SaveEntry(e)
	if err != nil {
		return err
	}

	err = s.core.Build(e.Deleted())
	if err != nil {
		return err
	}

	go s.postSaveEntry(e, options)
	return nil
}

func (s *Server) preSaveEntry(e *core.Entry) error {
	s.log.Infow("pre save entry hooks", "id", e.ID)

	for name, plugin := range s.plugins {
		hookPlugin, ok := plugin.(HookPlugin)
		if !ok {
			continue
		}

		err := hookPlugin.PreSaveHook(e)
		if err != nil {
			return fmt.Errorf("plugin %s error: %w", name, err)
		}
	}

	return nil
}

type postSaveEntryOptions struct {
	isNew             bool
	skipBuild         bool
	syndicators       []string
	syndicationStatus string
	previousLinks     []string
}

func (s *Server) postSaveEntry(e *core.Entry, options postSaveEntryOptions) {
	s.log.Infow("post save entry hooks", "id", e.ID)

	// Syndications
	syndicators, syndicationStatus := s.updateSchedule(e, options.syndicators, options.syndicationStatus)
	err := s.syndicate(e, syndicators, syndicationStatus)
	if err != nil {
		s.log.Errorw("errors occurred while syndicating", "id", e.ID, "err", err)
	}

	// Post-save hooks
	for name, plugin := range s.plugins {
		hookPlugin, ok := plugin.(HookPlugin)
		if !ok {
			continue
		}

		err := hookPlugin.PostSaveHook(e, options.isNew)
		if err != nil {
			s.log.Errorw("plugin post save hook failed", "plugin", name, "err", err)
		}
	}

	// Search indexing
	if s.meilisearch != nil {
		var err error
		if e.Deleted() {
			err = s.meilisearch.Remove(e.ID)
		} else {
			err = s.meilisearch.Add(e)
		}
		if err != nil {
			s.log.Errorw("meilisearch sync failed", "err", err)
		}
	}

	// Rebuild
	if !options.skipBuild && !e.Deleted() && !e.Draft && !e.Future() {
		s.build(false)
	}

	err = s.core.SendWebmentions(e, options.previousLinks...)
	if err != nil {
		s.log.Errorw("failed to send webmentions", "id", e.ID, "err", err)
	}
}

func (s *Server) updateSchedule(e *core.Entry, syndicators []string, status string) ([]string, string) {
	ctx := context.Background()
	db := s.core.DB()

	scheduled, err := db.GetScheduledEntry(ctx, e.ID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.log.Errorw("failed to get scheduled entry", "id", e.ID, "err", err)
		}
		scheduled = nil
	}

	switch {
	case e.Deleted() || e.Draft:
		if scheduled != nil {
			err = db.DeleteScheduledEntry(ctx, e.ID)
			if err != nil {
				s.log.Errorw("failed to delete scheduled entry", "id", e.ID, "err", err)
			}
		}
		return nil, ""
	case e.Future():
		if scheduled == nil {
			scheduled = &core.ScheduledEntry{ID: e.ID}
		}
		scheduled.Date = e.Date
		scheduled.Syndicators = append(scheduled.Syndicators, syndicators...)
		// Existing syndications are deleted by syndicate. Re-create them once the
		// entry is published.
		for name, syndicator := range s.syndicators {
			if syndicator.IsSyndicated(e) {
				scheduled.Syndicators = append(scheduled.Syndicators, name)
			}
		}
		scheduled.Syndicators = lo.Uniq(scheduled.Syndicators)
		sort.Strings(scheduled.Syndicators)
		if status != "" {
			scheduled.SyndicationStatus = status
		}

		err = db.SaveScheduledEntry(ctx, scheduled)
		if err != nil {
			s.log.Errorw("failed to save scheduled entry", "id", e.ID, "err", err)
		}
		return nil, ""
	default:
		if scheduled == nil {
			return syndicators, status
		}

		err = db.DeleteScheduledEntry(ctx, e.ID)
		if err != nil {
			s.log.Errorw("failed to delete scheduled entry", "id", e.ID, "err", err)
		}
		if status == "" {
			status = scheduled.SyndicationStatus
		}
		return append(syndicators, scheduled.Syndicators...), status
	}
}

func (s *Server) publishScheduled() {
	entries, err := s.core.DB().GetDueScheduledEntries(context.Background(), time.Now())
	if err != nil {
		s.log.Errorw("failed to get due scheduled entries", "err", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	s.log.Infow("publishing scheduled entries", "n", len(entries))
	s.build(false)

	for _, scheduled := range entries {
		e, err := s.core.GetEntry(scheduled.ID)
		if os.IsNotExist(err) {
			err = s.core.DB().DeleteScheduledEntry(context.Background(), scheduled.ID)
			if err != nil {
				s.log.Errorw("failed to delete scheduled entry", "id", scheduled.ID, "err", err)
			}
			continue
		} else if err != nil {
			s.log.Errorw("failed to open scheduled entry", "id", scheduled.ID, "err", err)
			continue
		}

		// updateSchedule merges the stored syndicators.
		s.postSaveEntry(e, postSaveEntryOptions{skipBuild: true})
		time.Sleep(time.Second)
	}
}
