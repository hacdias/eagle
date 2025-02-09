package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/gabriel-vasile/mimetype"
	"github.com/karlseguin/typed"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/services/media"
	"go.hacdias.com/indielib/micropub"
)

const (
	micropubPath = "/micropub"

	SyndicationField = "syndication"
)

func (s *Server) getMicropubChannels() []micropub.Channel {
	taxons, err := s.bolt.GetTaxonomy(context.Background(), s.c.Micropub.ChannelsTaxonomy)
	if err != nil {
		s.log.Errorw("failed to fetch channels taxonomy", "taxonomy", s.c.Micropub.ChannelsTaxonomy, "err", err)
		return nil
	}

	return lo.Map(taxons, func(t string, _ int) micropub.Channel {
		return micropub.Channel{
			UID:  t,
			Name: t,
		}
	})
}

func (s *Server) getMicropubCategories() []string {
	taxons, err := s.bolt.GetTaxonomy(context.Background(), s.c.Micropub.CategoriesTaxonomy)
	if err != nil {
		s.log.Errorw("failed to fetch categories taxonomy", "taxonomy", s.c.Micropub.CategoriesTaxonomy, "err", err)
		return nil
	}

	return taxons
}

func (s *Server) getMicropubSyndications() []micropub.Syndication {
	syndications := []micropub.Syndication{}
	for _, syndicator := range s.syndicators {
		syndications = append(syndications, syndicator.Syndication())
	}
	return syndications
}

func (s *Server) makeMicropub() http.Handler {
	var options []micropub.Option

	if len(s.c.Micropub.PostTypes) != 0 {
		options = append(options, micropub.WithGetPostTypes(func() []micropub.PostType {
			return s.c.Micropub.PostTypes
		}))
	}

	if s.c.Micropub.ChannelsTaxonomy != "" {
		options = append(options, micropub.WithGetChannels(s.getMicropubChannels))
	}

	if s.c.Micropub.CategoriesTaxonomy != "" {
		options = append(options, micropub.WithGetCategories(s.getMicropubCategories))
	}

	if s.media != nil {
		options = append(options, micropub.WithMediaEndpoint(s.c.AbsoluteURL(micropubMediaPath)))
	}

	options = append(options, micropub.WithGetSyndicateTo(s.getMicropubSyndications))

	return micropub.NewHandler(&micropubServer{
		s: s,
	}, options...)
}

type micropubServer struct {
	s *Server
}

func (m *micropubServer) HasScope(r *http.Request, scope string) bool {
	return lo.Contains(m.s.getScopes(r), scope)
}

func (m *micropubServer) Source(url string) (map[string]any, error) {
	e, err := m.s.core.GetEntryFromPermalink(url)
	if err != nil {
		return nil, err
	}

	return m.entryToMF2(e), nil
}

func (m *micropubServer) SourceMany(limit, offset int) ([]map[string]any, error) {
	return nil, micropub.ErrNotImplemented
}

func (m *micropubServer) Create(req *micropub.Request) (string, error) {
	slug := getRequestSlug(req)
	if slug == "" {
		return "", fmt.Errorf("%w: mp-slug is missing", micropub.ErrBadRequest)
	}

	id := core.NewPostID(slug, time.Now())
	e := m.s.core.NewBlankEntry(id)

	err := m.updateEntryWithProps(e, req.Properties)
	if err != nil {
		return "", err
	}

	if e.Title == "" {
		return "", fmt.Errorf("%w: name is missing", micropub.ErrBadRequest)
	}

	if m.s.c.Micropub.ChannelsTaxonomy != "" {
		e.Other[m.s.c.Micropub.ChannelsTaxonomy], _ = getRequestChannels(req)
	}

	err = m.preSave(e)
	if err != nil {
		return "", err
	}

	err = m.s.core.SaveEntry(e)
	if err != nil {
		return "", err
	}

	err = m.s.core.Build(false)
	if err != nil {
		return "", err
	}

	go m.postSave(e, req, nil)
	return e.Permalink, nil
}

func (m *micropubServer) Update(req *micropub.Request) (string, error) {
	return req.URL, m.update(req.URL, req, func(e *core.Entry) (error, bool) {
		mf2 := m.entryToMF2(e)["properties"].(map[string][]any)
		mf2, err := Update(mf2, req.Updates)
		if err != nil {
			return err, false
		}

		if m.s.c.Micropub.ChannelsTaxonomy != "" {
			channels, set := getRequestChannels(req)
			if set {
				e.Other[m.s.c.Micropub.ChannelsTaxonomy] = channels
			}
		}

		e.Lastmod = time.Now()
		return m.updateEntryWithProps(e, mf2), true
	})
}

func (m *micropubServer) Delete(url string) error {
	return m.update(url, nil, func(e *core.Entry) (error, bool) {
		if e.Deleted() {
			return nil, false
		}

		e.ExpiryDate = time.Now()
		return nil, true
	})
}

func (m *micropubServer) Undelete(url string) error {
	return m.update(url, nil, func(e *core.Entry) (error, bool) {
		if !e.Deleted() {
			return nil, false
		}

		e.ExpiryDate = time.Time{}
		return nil, true
	})
}

func (m *micropubServer) update(permalink string, req *micropub.Request, update func(e *core.Entry) (error, bool)) error {
	targets, _ := m.s.core.GetEntryLinks(permalink, true)

	e, err := m.s.core.GetEntryFromPermalink(permalink)
	if err != nil {
		return err
	}

	err, modified := update(e)
	if err != nil {
		return err
	}

	if !modified {
		return nil
	}

	err = m.preSave(e)
	if err != nil {
		return err
	}

	err = m.s.core.SaveEntry(e)
	if err != nil {
		return err
	}

	err = m.s.core.Build(e.Deleted())
	if err != nil {
		return err
	}

	go m.postSave(e, req, targets)
	return nil
}

func (m *micropubServer) getPhotos(e *core.Entry) ([]Photo, error) {
	var photos []Photo

	for i, photo := range typed.New(e.Other).Objects("photos") {
		if i >= 4 {
			break
		}

		photoUrl := photo.String("url")
		if photoUrl == "" {
			return nil, errors.New("photo has no url")
		}

		photoUrl, err := m.s.media.GetImageURL(photoUrl, media.FormatJPEG, media.Width1000)
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
			return nil, fmt.Errorf("cannot detect mimetype of %s", photo)
		}

		photos = append(photos, Photo{
			Data:     data,
			MimeType: mime.String(),
		})
	}

	return photos, nil
}

func (m *micropubServer) syndicate(e *core.Entry, syndicators []string) {
	// Get the photos to use during syndication
	photos, err := m.getPhotos(e)
	if err != nil {
		m.s.log.Errorw("failed to get photos for syndication", "entry", e.ID, "err", err)
		return
	}

	// Include syndicators that have already been used for this post
	for name, syndicator := range m.s.syndicators {
		if syndicator.IsSyndicated(e) {
			syndicators = append(syndicators, name)
		}
	}

	// Do the actual syndication
	syndications := typed.New(e.Other).Strings(SyndicationField)
	for _, name := range syndicators {
		if syndicator, ok := m.s.syndicators[name]; ok {
			syndication, removed, err := syndicator.Syndicate(context.Background(), e, photos)
			if err != nil {
				m.s.log.Errorw("failed to syndicate", "entry", e.ID, "syndicator", name, "err", err)
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

	err = m.s.core.SaveEntry(e)
	if err != nil {
		m.s.log.Errorw("failed save entry", "id", e.ID, "err", err)
	}
}

func (m *micropubServer) preSave(e *core.Entry) error {
	for name, plugin := range m.s.plugins {
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

func (m *micropubServer) postSave(e *core.Entry, req *micropub.Request, oldTargets []string) {
	// Syndications
	var syndicateTo []string
	if req != nil {
		syndicateTo, _ = getRequestSyndicateTo(req)
	}
	m.syndicate(e, syndicateTo)

	// Post-save hooks
	for name, plugin := range m.s.plugins {
		hookPlugin, ok := plugin.(HookPlugin)
		if !ok {
			continue
		}

		err := hookPlugin.PostSaveHook(e)
		if err != nil {
			m.s.log.Errorw("plugin post save hook failed", "plugin", name, "err", err)
		}
	}

	// Search indexing
	if m.s.meilisearch != nil {
		var err error
		if e.Deleted() {
			err = m.s.meilisearch.Remove(e.ID)
		} else {
			err = m.s.meilisearch.Add(e)
		}
		if err != nil {
			m.s.log.Errorw("meilisearch sync failed", "err", err)
		}
	}

	// Rebuild
	m.s.build(false)

	// No further action for drafts or no webmentions
	if e.Draft || e.NoWebmentions {
		return
	}

	err := m.s.core.SendWebmentions(e.Permalink, oldTargets...)
	if err != nil {
		m.s.log.Errorw("failed to send webmentions", "id", e.ID, "err", err)
	}
}

func (m *micropubServer) entryToMF2(e *core.Entry) map[string]any {
	properties := map[string]interface{}{}

	for _, k := range m.s.c.Micropub.Properties {
		if v, ok := e.Other[k]; ok {
			properties[k] = v
		}
	}

	if !e.Date.IsZero() {
		properties["published"] = e.Date.Format(time.RFC3339)
	}

	if !e.Lastmod.IsZero() {
		properties["updated"] = e.Lastmod.Format(time.RFC3339)
	}

	properties["content"] = e.Content

	if e.Title != "" {
		properties["name"] = e.Title
	}

	if e.Description != "" {
		properties["summary"] = e.Description
	}

	if e.Draft {
		properties["post-status"] = "draft"
	} else if e.Deleted() {
		properties["post-status"] = "deleted"
	} else {
		properties["post-status"] = "published"
	}

	if m.s.c.Micropub.CategoriesTaxonomy != "" {
		taxons := e.Taxonomy(m.s.c.Micropub.CategoriesTaxonomy)
		if len(taxons) != 0 {
			properties["category"] = e.Taxonomy(m.s.c.Micropub.CategoriesTaxonomy)
		}
	}

	if m.s.c.Micropub.ChannelsTaxonomy != "" {
		properties["mp-channel"] = e.Taxonomy(m.s.c.Micropub.ChannelsTaxonomy)
	}

	return Deflatten(map[string]interface{}{
		"type":       "h-entry",
		"properties": properties,
	})
}

func (m *micropubServer) updateEntryWithProps(e *core.Entry, newProps map[string][]interface{}) error {
	properties := typed.New(Flatten(newProps))

	// Micropublish.net sends the file name that was uploaded through
	// the media endpoint as a property. This is unnecessary.
	delete(properties, "file")

	if e.Other == nil {
		e.Other = map[string]any{}
	}

	if published, ok := properties.StringIf("published"); ok {
		p, err := dateparse.ParseStrict(published)
		if err != nil {
			return err
		}
		e.Date = p
		delete(properties, "published")
	}

	if updated, ok := properties.StringIf("updated"); ok {
		p, err := dateparse.ParseStrict(updated)
		if err != nil {
			return err
		}
		e.Lastmod = p
		delete(properties, "updated")
	}

	if content, ok := properties.StringIf("content"); ok {
		e.Content = content
		delete(properties, "content")
	} else if content, ok := properties.ObjectIf("content"); ok {
		if text, ok := content.StringIf("text"); ok {
			e.Content = text
		} else if html, ok := content.StringIf("html"); ok {
			e.Content = html
		} else {
			return errors.New("could not parse content field")
		}
	} else if _, ok := properties["content"]; ok {
		return errors.New("could not parse content field")
	}

	e.Content = strings.TrimSpace(e.Content)

	if name, ok := properties.StringIf("name"); ok {
		e.Title = name
		delete(properties, "name")
	}

	if summary, ok := properties.StringIf("summary"); ok {
		e.Description = summary
		delete(properties, "summary")
	}

	if status, ok := properties.StringIf("post-status"); ok {
		if status == "draft" {
			e.Draft = true
		}
		delete(properties, "post-status")
	}

	if m.s.c.Micropub.CategoriesTaxonomy != "" {
		if categories, ok := properties.StringsIf("category"); ok && len(categories) > 0 {
			e.Other[m.s.c.Micropub.CategoriesTaxonomy] = categories
			delete(properties, "category")
		} else if category, ok := properties.StringIf("category"); ok && category != "" {
			e.Other[m.s.c.Micropub.CategoriesTaxonomy] = []string{category}
			delete(properties, "category")
		}
	}

	err := m.updateEntryWithPhotos(e, properties)
	if err != nil {
		return err
	}

	for _, key := range m.s.c.Micropub.Properties {
		// NOTE: this ensures that things that were arrays stay as arrays. However,
		// I should probably improve this such that there is a list of properties
		// to keep as arrays and others as non-arrays.
		// TODO: Maybe indielib/microformats.propertyToType properties should always be
		// single, the rest always arrays.
		if newValue, ok := properties[key]; ok {
			if oldValue, ok := e.Other[key]; ok {
				oldKind := reflect.TypeOf(oldValue).Kind()
				newKind := reflect.TypeOf(newValue).Kind()
				if oldKind == reflect.Slice && newKind != reflect.Slice {
					e.Other[key] = []any{newValue}
				} else {
					e.Other[key] = newValue
				}
			} else {
				e.Other[key] = newValue
			}
			delete(properties, key)
		}
	}

	// Get remaining keys, except mp- commands
	keys := lo.Filter(lo.Keys(properties), func(prop string, index int) bool {
		return !strings.HasPrefix(prop, "mp-")
	})
	if len(keys) > 0 {
		return fmt.Errorf("unknown keys: %s", strings.Join(keys, ", "))
	}

	return nil
}

func (m *micropubServer) updateEntryWithPhotos(e *core.Entry, properties typed.Typed) error {
	parts := strings.Split(strings.TrimSuffix(e.ID, "/"), "/")
	slug := parts[len(parts)-1]
	prefix := fmt.Sprintf("%04d-%02d-%s", e.Date.Year(), e.Date.Month(), slug)

	photoUrls := []string{}
	photoData := map[string][]byte{}

	if url, ok := properties.StringIf("photo"); ok {
		data, ok := m.s.mediaCache.Get(url)
		if !ok {
			return fmt.Errorf("photo %q not found in cache", url)
		}

		photoUrls = append(photoUrls, url)
		photoData[url] = data
		m.s.mediaCache.Delete(url)

		delete(properties, "photo")
	} else if photos, ok := properties.StringsIf("photo"); ok {
		for _, url := range photos {
			data, ok := m.s.mediaCache.Get(url)
			if !ok {
				return fmt.Errorf("photo %q not found in cache", url)
			}

			photoUrls = append(photoUrls, url)
			photoData[url] = data
			m.s.mediaCache.Delete(url)
		}

		delete(properties, "photo")
	}

	if len(photoUrls) == 0 {
		return nil
	}

	photos := []any{}

	for i, url := range photoUrls {
		data := photoData[url]
		filename := prefix
		if len(photoUrls) > 1 {
			filename += fmt.Sprintf("-%02d", i+1)
		}

		ext := filepath.Ext(url)
		cdnUrl, err := m.s.media.UploadMedia(filename, ext, bytes.NewBuffer(data))
		if err != nil {
			return fmt.Errorf("failed to upload photo: %w", err)
		}

		photos = append(photos, map[string]string{
			"url": cdnUrl,
		})
	}

	e.Other["photos"] = photos

	return nil
}
