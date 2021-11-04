package eagle

import (
	"errors"
	"fmt"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/v2/pkg/mf2"
	"github.com/karlseguin/typed"
	"github.com/thoas/go-funk"
)

var allowedLetters = []rune("abcdefghijklmnopqrstuvwxyz")

func (e *Eagle) NewPostID(slug string, t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}

	if slug == "" {
		slug = funk.RandomString(5, allowedLetters)
	}

	return fmt.Sprintf("/%04d/%02d/%02d/%s", t.Year(), t.Month(), t.Day(), slug)
}

func (e *Eagle) EntryFromMF2(mf2 map[string][]interface{}, slug string) (*Entry, error) {
	entry := &Entry{
		Frontmatter: Frontmatter{},
	}

	err := e.UpdateEntryWithMF2(entry, mf2)
	if err != nil {
		return nil, err
	}

	id := e.NewPostID(slug, entry.Published)

	entry.ID = e.cleanID(id)
	entry.Permalink, err = e.makePermalink(id)

	return entry, err
}

func (e *Eagle) UpdateEntryWithMF2(entry *Entry, mf2Data map[string][]interface{}) error {
	data := typed.New(mf2.Flatten(mf2Data))
	postType, _ := mf2.DiscoverType(data)

	if funk.Contains(e.allowedTypes, postType) {
		if entry.Section == "" {
			entry.Section = e.Config.Site.MicropubTypes[postType]
		}
	} else {
		return errors.New("type not supported " + string(postType))
	}

	if published, ok := data.StringIf("published"); ok {
		p, err := dateparse.ParseStrict(published)
		if err != nil {
			return err
		}
		entry.Published = p
		delete(data, "published")
	} else {
		entry.Published = time.Now()
	}

	if updated, ok := data.StringIf("updated"); ok {
		p, err := dateparse.ParseStrict(updated)
		if err != nil {
			return err
		}
		entry.Updated = p
		delete(data, "updated")
	}

	if content, ok := data.StringIf("content"); ok {
		entry.Content = content
		delete(data, "content")
	} else if _, ok := data["content"]; ok {
		return errors.New("could not parse content field")
	}

	if name, ok := data.StringIf("name"); ok {
		entry.Title = name
		delete(data, "name")
	}

	if summary, ok := data.StringIf("summary"); ok {
		entry.Description = summary
		delete(data, "summary")
	}

	if status, ok := data.StringIf("post-status"); ok {
		if status == "draft" {
			entry.Draft = true
		}
		delete(data, "post-status")
	}

	if visibility, ok := data.StringIf("visibility"); ok {
		if visibility == "private" {
			entry.Private = true
		}
		delete(data, "visibility")
	}

	if entry.Properties == nil {
		entry.Properties = map[string]interface{}{}
	}

	entry.Properties = data
	return nil
}

func (e *Entry) ToFlatMF2() map[string]interface{} {
	// Shallow copy of the map because we are not changing
	// the values inside.
	properties := map[string]interface{}{}
	for k, v := range e.Properties {
		properties[k] = v
	}

	if !e.Published.IsZero() {
		properties["published"] = e.Published.Format(time.RFC3339)
	}

	if !e.Updated.IsZero() {
		properties["updated"] = e.Updated.Format(time.RFC3339)
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
	} else {
		properties["post-status"] = "published"
	}

	if e.Private {
		properties["visibility"] = "private"
	} else {
		properties["visibility"] = "public"
	}

	return map[string]interface{}{
		"type":       []string{"h-entry"},
		"properties": properties,
	}
}

func (e *Entry) ToMF2() map[string]interface{} {
	return mf2.Deflatten(e.ToFlatMF2())
}
