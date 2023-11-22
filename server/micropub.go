package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
)

func (s *Server) makeMicropub() http.Handler {
	var options []micropub.Option

	if len(s.c.Micropub.PostTypes) != 0 {
		options = append(options, micropub.WithGetPostTypes(func() []micropub.PostType {
			return s.c.Micropub.PostTypes
		}))
	}

	if s.c.Micropub.ChannelsTaxonomy != "" {
		options = append(options, micropub.WithGetChannels(func() []micropub.Channel {
			taxons, err := s.bolt.GetTaxonomy(context.Background(), s.c.Micropub.ChannelsTaxonomy)
			if err != nil {
				s.log.Warnw("could not fetch channels taxonomy", "taxonomy", s.c.Micropub.ChannelsTaxonomy, "err", err)
				return nil
			}

			return lo.Map(taxons, func(t string, _ int) micropub.Channel {
				return micropub.Channel{
					UID:  t,
					Name: t,
				}
			})
		}))
	}

	if s.c.Micropub.CategoriesTaxonomy != "" {
		options = append(options, micropub.WithGetCategories(func() []string {
			taxons, err := s.bolt.GetTaxonomy(context.Background(), s.c.Micropub.CategoriesTaxonomy)
			if err != nil {
				s.log.Warnw("could not fetch categories taxonomy", "taxonomy", s.c.Micropub.CategoriesTaxonomy, "err", err)
				return nil
			}

			return taxons
		}))
	}

	return micropub.NewHandler(&micropubServer{s: s}, options...)
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
	slug := ""
	if slugs, ok := req.Commands["slug"]; ok {
		if len(slugs) == 1 {
			slug, _ = slugs[0].(string)
		}
	}
	if slug == "" {
		return "", fmt.Errorf("%w: mp-slug is missing", micropub.ErrBadRequest)
	}

	id := "/" + core.SpecialSection + "/" + time.Now().Format("2006") + "/" + slug + "/"
	e := m.s.core.NewBlankEntry(id)

	err := m.updateEntryWithProps(e, req.Properties)
	if err != nil {
		return "", err
	}

	if e.Title == "" {
		return "", fmt.Errorf("%w: name is missing", micropub.ErrBadRequest)
	}

	if m.s.c.Micropub.ChannelsTaxonomy != "" {
		var taxons []string
		if channels, ok := req.Commands["channel"]; ok {
			for _, ch := range channels {
				if v, ok := ch.(string); ok {
					taxons = append(taxons, v)
				}
			}
		}
		e.Other[m.s.c.Micropub.ChannelsTaxonomy] = taxons
	}

	err = m.s.core.SaveEntry(e)
	if err != nil {
		return "", err
	}

	go m.postRunActions(e, false, nil)
	return e.Permalink, nil
}

func (m *micropubServer) Update(req *micropub.Request) (string, error) {
	return req.URL, m.updateWithPostRun(req.URL, false, func(e *core.Entry) (error, bool) {
		mf2 := m.entryToMF2(e)["properties"].(map[string][]any)
		mf2, err := Update(mf2, req.Updates)
		if err != nil {
			return err, false
		}

		return m.updateEntryWithProps(e, mf2), true
	})
}

func (m *micropubServer) Delete(url string) error {
	return m.updateWithPostRun(url, true, func(e *core.Entry) (error, bool) {
		if e.Deleted() {
			return nil, false
		}

		e.ExpiryDate = time.Now()
		return nil, true
	})
}

func (m *micropubServer) Undelete(url string) error {
	return m.updateWithPostRun(url, false, func(e *core.Entry) (error, bool) {
		if !e.Deleted() {
			return nil, false
		}

		e.ExpiryDate = time.Time{}
		return nil, true
	})
}

func (m *micropubServer) updateWithPostRun(permalink string, clean bool, update func(e *core.Entry) (error, bool)) error {
	targets, _ := m.s.core.GetEntryLinks(permalink)

	e, err := m.s.core.GetEntryFromPermalink(permalink)
	if err != nil {
		return err
	}

	err, ok := update(e)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	err = m.s.core.SaveEntry(e)
	if err != nil {
		return err
	}

	go m.postRunActions(e, clean, targets)
	return nil
}

func (m *micropubServer) postRunActions(e *core.Entry, cleanBuild bool, oldTargets []string) {
	var err error

	if m.s.meilisearch != nil {
		if e.Deleted() {
			err = m.s.meilisearch.Remove(e.ID)
		} else {
			err = m.s.meilisearch.Add(e)
		}
		if err != nil {
			m.s.n.Error(fmt.Errorf("meilisearch sync failed: %w", err))
		}
	}

	m.s.buildNotify(cleanBuild)

	if e.Draft || e.NoWebmentions {
		return
	}

	err = m.s.core.SendWebmentions(e.Permalink, oldTargets...)
	if err != nil {
		m.s.n.Error(fmt.Errorf("meilisearch sync failed: %w", err))
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

	keys := lo.Keys(properties)
	if len(keys) > 0 {
		return fmt.Errorf("unknown keys: %s", strings.Join(keys, ", "))
	}

	return nil
}
