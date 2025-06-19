package server

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"net/http"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/gabriel-vasile/mimetype"
	"github.com/karlseguin/typed"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/services/media"
	"go.hacdias.com/indielib/micropub"
)

func (s *Server) getPhoto(url string) (*Photo, error) {
	photoUrl, err := s.media.GetImageURL(url, media.FormatJPEG, media.Width1800)
	if err != nil {
		return nil, err
	}

	res, err := http.Get(photoUrl)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	mime := mimetype.Detect(data)
	if mime == nil {
		return nil, fmt.Errorf("cannot detect mimetype of %s", url)
	}

	return &Photo{
		Data:     data,
		MimeType: mime.String(),
	}, nil
}

func (s *Server) getEntrySyndicationContext(e *core.Entry) (*SyndicationContext, error) {
	ctx := &SyndicationContext{}

	thumbnailStr := typed.New(e.Other).String("thumbnail")

	// Get the first 4 photos from the entry
	for i, p := range e.Photos {
		if i >= 4 {
			break
		}

		photo, err := s.getPhoto(p.URL)
		if err != nil {
			return nil, err
		}
		photo.Title = p.Title
		photo.Width = p.Width
		photo.Height = p.Height

		if p.URL == thumbnailStr {
			ctx.Thumbnail = photo
		}

		ctx.Photos = append(ctx.Photos, photo)
	}

	if ctx.Thumbnail == nil && thumbnailStr != "" {
		var err error
		ctx.Thumbnail, err = s.getPhoto(thumbnailStr)
		if err != nil {
			return nil, err
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

	// Do the actual syndication
	syndications := typed.New(e.Other).Strings(SyndicationField)
	for _, name := range syndicators {
		if syndicator, ok := s.syndicators[name]; ok {
			syndication, removed, err := syndicator.Syndicate(context.Background(), e, syndicationContext)
			if err != nil {
				s.log.Errorw("failed to syndicate", "entry", e.ID, "syndicator", name, "err", err)
				continue
			}

			if removed {
				syndications = lo.Without(syndications, syndication)
			} else {
				syndications = append(syndications, syndication)
			}
		}
	}

	syndications = lo.Uniq(syndications)
	if len(syndications) == 0 {
		delete(e.Other, SyndicationField)
	} else {
		e.Other[SyndicationField] = lo.Uniq(syndications)
	}

	err = s.core.SaveEntry(e)
	if err != nil {
		s.log.Errorw("failed save entry", "id", e.ID, "err", err)
	}
}

func (s *Server) preSaveEntry(e *core.Entry) error {
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

func (s *Server) postSaveEntry(e *core.Entry, req *micropub.Request, oldTargets []string, skipBuild bool) {
	// Syndications
	var syndicateTo []string
	if req != nil {
		syndicateTo, _ = getRequestSyndicateTo(req)
	}
	s.syndicate(e, syndicateTo)

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
	if !skipBuild && !e.Deleted() && !e.Draft {
		s.build(false)
	}

	// No further action for drafts or no webmentions
	if e.Draft || e.NoWebmentions {
		return
	}

	err := s.core.SendWebmentions(e.Permalink, oldTargets...)
	if err != nil {
		s.log.Errorw("failed to send webmentions", "id", e.ID, "err", err)
	}
}
