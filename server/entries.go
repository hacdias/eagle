package server

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"sort"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/karlseguin/typed"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
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

func (s *Server) syndicate(e *core.Entry, syndicators []string) {
	if !e.IsPost() {
		return
	}

	// Get the syndication context
	syndicationContext, err := s.getEntrySyndicationContext(e)
	if err != nil {
		s.log.Errorw("failed to get syndication context", "entry", e.ID, "err", err)
		return
	}

	// Include syndicators that have already been used for this post
	for name, syndicator := range s.syndicators {
		if syndicator.IsSyndicated(e) {
			syndicators = append(syndicators, name)
		}
	}

	syndicators = lo.Uniq(syndicators)
	s.log.Infow("syndicating entry", "id", e.ID, "syndicators", syndicators)

	// Do the actual syndication
	for _, name := range syndicators {
		if syndicator, ok := s.syndicators[name]; ok {
			err := syndicator.Syndicate(context.Background(), e, syndicationContext)
			if err != nil {
				s.log.Errorw("failed to syndicate", "entry", e.ID, "syndicator", name, "err", err)
				continue
			}
		}
	}

	// Ensure uniqueness and that it always yields the same result
	e.Syndications = lo.Uniq(e.Syndications)
	sort.Strings(e.Syndications)

	err = s.core.SaveEntry(e)
	if err != nil {
		s.log.Errorw("failed save entry", "id", e.ID, "err", err)
	}

	s.log.Infow("syndicated entry", "id", e.ID)
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
	skipBuild     bool
	syndicators   []string
	previousLinks []string
}

func (s *Server) postSaveEntry(e *core.Entry, options postSaveEntryOptions) {
	s.log.Infow("post save entry hooks", "id", e.ID)

	// Syndications
	s.syndicate(e, options.syndicators)

	// Post-save hooks
	for name, plugin := range s.plugins {
		hookPlugin, ok := plugin.(HookPlugin)
		if !ok {
			continue
		}

		err := hookPlugin.PostSaveHook(e)
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
	if !options.skipBuild && !e.Deleted() && !e.Draft {
		s.build(false)
	}

	err := s.core.SendWebmentions(e, options.previousLinks...)
	if err != nil {
		s.log.Errorw("failed to send webmentions", "id", e.ID, "err", err)
	}
}
