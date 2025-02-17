package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/karlseguin/typed"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
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

	err = m.s.preSaveEntry(e)
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

	go m.s.postSaveEntry(e, req, nil, false)
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

	err = m.s.preSaveEntry(e)
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

	go m.s.postSaveEntry(e, req, targets, false)
	return nil
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

	if len(e.Photos) > 0 {
		properties["photo"] = lo.Map(e.Photos, func(p core.Photo, i int) string {
			return p.URL
		})
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
		m.s.log.Warnw("unknown micropub keys", "keys", keys)
	}

	return nil
}

func (m *micropubServer) updateEntryWithPhotos(e *core.Entry, properties typed.Typed) error {
	// Define prefix for the photos that will be uploaded
	parts := strings.Split(strings.TrimSuffix(e.ID, "/"), "/")
	slug := parts[len(parts)-1]
	prefix := fmt.Sprintf("%04d-%02d-%s", e.Date.Year(), e.Date.Month(), slug)

	urls := []string{}
	cachedData := map[string][]byte{}

	if url, ok := properties.StringIf("photo"); ok {
		if strings.HasPrefix(url, "cache:/") {
			data, ok := m.s.mediaCache.Get(url)
			if !ok {
				return fmt.Errorf("photo %q not found in cache", url)
			}

			cachedData[url] = data
			m.s.mediaCache.Delete(url)
		}

		urls = append(urls, url)
		delete(properties, "photo")
	} else if photos, ok := properties.StringsIf("photo"); ok {
		for _, url := range photos {
			if strings.HasPrefix(url, "cache:/") {
				data, ok := m.s.mediaCache.Get(url)
				if !ok {
					return fmt.Errorf("photo %q not found in cache", url)
				}

				cachedData[url] = data
				m.s.mediaCache.Delete(url)
			}

			urls = append(urls, url)
		}

		delete(properties, "photo")
	}

	if len(urls) == 0 {
		e.Photos = nil
		return nil
	}

	// Get old titles
	titles := lo.Reduce(e.Photos, func(t map[string]string, p core.Photo, i int) map[string]string {
		t[p.Title] = p.URL
		return t
	}, map[string]string{})

	e.Photos = []core.Photo{}
	for i, url := range urls {
		data, isCached := cachedData[url]

		if isCached {
			filename := prefix
			if len(urls) > 1 {
				filename += fmt.Sprintf("-%02d", i+1)
			}

			ext := filepath.Ext(url)
			cdnUrl, err := m.s.media.UploadMedia(filename, ext, bytes.NewBuffer(data))
			if err != nil {
				return fmt.Errorf("failed to upload photo: %w", err)
			}

			e.Photos = append(e.Photos, core.Photo{
				URL: cdnUrl,
			})
		} else {
			e.Photos = append(e.Photos, core.Photo{
				URL:   url,
				Title: titles[url],
			})
		}
	}

	return nil
}
