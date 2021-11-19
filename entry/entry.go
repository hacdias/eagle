package entry

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/v2/entry/mf2"
	"github.com/hacdias/eagle/v2/util"
	"github.com/karlseguin/typed"
	"github.com/microcosm-cc/bluemonday"
	"github.com/thoas/go-funk"
	stripMarkdown "github.com/writeas/go-strip-markdown"
	yaml "gopkg.in/yaml.v2"
)

type Entry struct {
	Frontmatter
	ID        string
	Permalink string
	Content   string

	helper  *mf2.FlatHelper
	summary string
}

func (e *Entry) Helper() *mf2.FlatHelper {
	if e.helper == nil {
		e.helper = mf2.NewFlatHelper(e.FlatMF2())
	}

	return e.helper
}

func (e *Entry) Tags() []string {
	m := typed.New(e.Properties)

	if v, ok := m.StringIf("category"); ok {
		return []string{v}
	}

	// Slight modification of StringsIf so we capture
	// all string elements instead of blocking when there is none.
	// Tags can also be objects, such as tagged people as seen in
	// here: https://ownyourswarm.p3k.io/docs
	value, ok := m["category"]
	if !ok {
		return []string{}
	}

	if n, ok := value.([]string); ok {
		return n
	}

	if a, ok := value.([]interface{}); ok {
		n := []string{}
		for i := 0; i < len(a); i++ {
			if v, ok := a[i].(string); ok {
				n = append(n, v)
			}
		}
		return n
	}

	return []string{}
}

func (e *Entry) Summary() string {
	if e.summary != "" {
		return e.summary
	}

	if strings.Contains(e.Content, "<!--more-->") {
		firstPart := strings.Split(e.Content, "<!--more-->")[0]
		e.summary = stripText(strings.TrimSpace(firstPart))
	} else if e.Description != "" {
		e.summary = e.Description
	} else if content := e.TextContent(); content != "" {
		e.summary = util.TruncateString(content, 300) + "…"
	}

	// TODO: get context and trim that text.
	return e.summary
}

func (e *Entry) InSection(section string) bool {
	return funk.ContainsString(e.Sections, section)
}

func (e *Entry) String() (string, error) {
	val, err := yaml.Marshal(&e.Frontmatter)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n\n%s\n", string(val), e.Content), nil
}

func (e *Entry) TextContent() string {
	return stripText(e.Content)
}

func (e *Entry) Update(mf2Data map[string][]interface{}) error {
	flattened := mf2.Flatten(mf2Data)
	data := typed.New(flattened)
	mm := mf2.NewFlatHelper(flattened)

	if published, ok := data.StringIf("published"); ok {
		p, err := dateparse.ParseStrict(published)
		if err != nil {
			return err
		}
		e.Published = p
		delete(data, "published")
	}

	if updated, ok := data.StringIf("updated"); ok {
		p, err := dateparse.ParseStrict(updated)
		if err != nil {
			return err
		}
		e.Updated = p
		delete(data, "updated")
	}

	if content, ok := data.StringIf("content"); ok {
		e.Content = content
		delete(data, "content")
	} else if content, ok := data.ObjectIf("content"); ok {
		if text, ok := content.StringIf("text"); ok {
			e.Content = text
		} else if html, ok := content.StringIf("html"); ok {
			e.Content = html
		} else {
			return errors.New("could not parse content field")
		}
	} else if _, ok := data["content"]; ok {
		return errors.New("could not parse content field")
	}

	e.Content = strings.TrimSpace(e.Content)

	if name, ok := data.StringIf("name"); ok {
		e.Title = name
		delete(data, "name")
	}

	if summary, ok := data.StringIf("summary"); ok {
		e.Description = summary
		delete(data, "summary")
	}

	if status, ok := data.StringIf("post-status"); ok {
		if status == "draft" {
			e.Draft = true
		}
		delete(data, "post-status")
	}

	if visibility, ok := data.StringIf("visibility"); ok {
		if visibility == "private" {
			e.Private = true
		}
		delete(data, "visibility")
	}

	if e.Properties == nil {
		e.Properties = map[string]interface{}{}
	}

	switch mm.PostType() {
	case mf2.TypeItinerary:
		if err := e.parseDateFromItinerary(data, mm); err != nil {
			return err
		}
	}

	if e.Published.IsZero() {
		e.Published = time.Now()
	}

	e.Properties = data
	return nil
}

func (e *Entry) parseDateFromItinerary(data typed.Typed, mm *mf2.FlatHelper) error {
	if !e.Published.IsZero() {
		return nil
	}

	itinerary, ok := data.ObjectIf(mm.TypeProperty())
	if !ok {
		return nil
	}

	props, ok := itinerary.ObjectIf("properties")
	if !ok {
		return nil
	}

	if arrival, ok := props.StringIf("arrival"); ok {
		p, err := dateparse.ParseStrict(arrival)
		if err != nil {
			return err
		}
		e.Published = p
	}

	return nil
}

func (e *Entry) FlatMF2() map[string]interface{} {
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
		"type":       "h-entry",
		"properties": properties,
	}
}

func (e *Entry) MF2() map[string]interface{} {
	return mf2.Deflatten(e.FlatMF2())
}

var htmlRemover = bluemonday.StrictPolicy()

func stripText(text string) string {
	text = htmlRemover.Sanitize(text)
	// Unescapes html entities.
	text = html.UnescapeString(text)
	text = stripMarkdown.Strip(text)
	return text
}
