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
	excerpt string
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

func (e *Entry) Visibility() Visibility {
	m := typed.New(e.Properties)
	switch m.String("visibility") {
	case "private":
		return VisibilityPrivate
	case "unlisted":
		return VisibilityUnlisted
	default:
		return VisibilityPublic
	}
}

func (e *Entry) Audience() []string {
	m := typed.New(e.Properties)

	if a := m.String("audience"); a != "" {
		return []string{a}
	}

	if aa := m.Strings("audience"); len(aa) != 0 {
		return aa
	}

	return nil
}

func (e *Entry) Excerpt() string {
	if e.excerpt != "" {
		return e.excerpt
	}

	if strings.Contains(e.Content, "<!--more-->") {
		firstPart := strings.Split(e.Content, "<!--more-->")[0]
		e.excerpt = stripText(strings.TrimSpace(firstPart))
	} else if content := e.TextContent(); content != "" {
		e.excerpt = util.TruncateString(content, 300) + "…"
	}

	return e.excerpt
}

func (e *Entry) DisplayTitle() string {
	if e.Title != "" {
		return e.Title
	}

	if e.Description != "" {
		return e.Description
	}

	excerpt := e.Excerpt()
	if excerpt == "" {
		return ""
	}

	if len(excerpt) > 100 {
		excerpt = strings.TrimSuffix(excerpt, "…")
		excerpt = util.TruncateString(excerpt, 100) + "…"
	}

	return excerpt
}

func (e *Entry) DisplayDescription() string {
	if e.Description != "" {
		return e.Description
	}

	return e.Excerpt()
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

func (e *Entry) Update(newProps map[string][]interface{}) error {
	props := typed.New(mf2.Flatten(newProps))
	mm := mf2.NewFlatHelper(props)
	e.Properties = props

	if published, ok := props.StringIf("published"); ok {
		p, err := dateparse.ParseStrict(published)
		if err != nil {
			return err
		}
		e.Published = p
		delete(props, "published")
	}

	if updated, ok := props.StringIf("updated"); ok {
		p, err := dateparse.ParseStrict(updated)
		if err != nil {
			return err
		}
		e.Updated = p
		delete(props, "updated")
	}

	if content, ok := props.StringIf("content"); ok {
		e.Content = content
		delete(props, "content")
	} else if content, ok := props.ObjectIf("content"); ok {
		if text, ok := content.StringIf("text"); ok {
			e.Content = text
		} else if html, ok := content.StringIf("html"); ok {
			e.Content = html
		} else {
			return errors.New("could not parse content field")
		}
	} else if _, ok := props["content"]; ok {
		return errors.New("could not parse content field")
	}

	e.Content = strings.TrimSpace(e.Content)

	if name, ok := props.StringIf("name"); ok {
		e.Title = name
		delete(props, "name")
	}

	if summary, ok := props.StringIf("summary"); ok {
		e.Description = summary
		delete(props, "summary")
	}

	if status, ok := props.StringIf("post-status"); ok {
		if status == "draft" {
			e.Draft = true
		}
		delete(props, "post-status")
	}

	switch mm.PostType() {
	case mf2.TypeItinerary:
		if err := e.parseDateFromItinerary(props, mm); err != nil {
			return err
		}

		// Make itineraries private if they're in the future.
		if e.Published.After(time.Now()) {
			e.Properties["visibility"] = VisibilityPrivate
		}
	}

	if e.Published.IsZero() {
		e.Published = time.Now().Local()
	}

	return nil
}

func (e *Entry) parseDateFromItinerary(data typed.Typed, mm *mf2.FlatHelper) error {
	if !e.Published.IsZero() {
		return nil
	}

	dateFromLeg := func(leg typed.Typed) error {
		props, ok := leg.ObjectIf("properties")
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

	if leg, ok := data.ObjectIf(mm.TypeProperty()); ok {
		return dateFromLeg(leg)
	} else if legs, ok := data.ObjectsIf(mm.TypeProperty()); ok {
		return dateFromLeg(legs[len(legs)-1])
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
