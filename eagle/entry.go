package eagle

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/util"
	"github.com/karlseguin/typed"
	"github.com/microcosm-cc/bluemonday"
	"github.com/samber/lo"
	stripMarkdown "github.com/writeas/go-strip-markdown"
	yaml "gopkg.in/yaml.v2"
)

const MoreSeparator = "<!--more-->"

type Entry struct {
	FrontMatter
	ID        string
	Permalink string
	Content   string

	helper      *mf2.FlatHelper
	excerpt     string
	textExcerpt string
}

func (e *Entry) Helper() *mf2.FlatHelper {
	if e.helper == nil {
		e.helper = mf2.NewFlatHelper(e.FlatMF2())
	}

	return e.helper
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

func (e *Entry) HasMore() bool {
	return strings.Contains(e.Content, MoreSeparator)
}

func (e *Entry) Excerpt() string {
	if e.excerpt != "" {
		return e.excerpt
	}

	if e.HasMore() {
		firstPart := strings.Split(e.Content, MoreSeparator)[0]
		e.excerpt = strings.TrimSpace(firstPart)
	} else if content := e.TextContent(); content != "" {
		e.excerpt = util.TruncateStringWithEllipsis(content, 300)
	}

	return e.excerpt
}

func (e *Entry) TextExcerpt() string {
	if e.textExcerpt != "" {
		return e.textExcerpt
	}

	e.textExcerpt = makePlainText(e.Excerpt())
	return e.textExcerpt
}

func (e *Entry) TextTitle() string {
	if e.Title != "" {
		return e.Title
	}

	if e.Description != "" {
		return e.Description
	}

	excerpt := e.TextExcerpt()
	if excerpt == "" {
		return ""
	}

	if len(excerpt) > 100 {
		excerpt = strings.TrimSuffix(excerpt, "…")
		excerpt = util.TruncateStringWithEllipsis(excerpt, 100)
	}

	return excerpt
}

func (e *Entry) TextDescription() string {
	if e.Description != "" {
		return e.Description
	}

	return e.TextExcerpt()
}

func (e *Entry) InSection(section string) bool {
	return lo.Contains(e.Sections, section)
}

func (e *Entry) String() (string, error) {
	val, err := yaml.Marshal(&e.FrontMatter)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n\n%s\n", string(val), strings.TrimSpace(e.Content)), nil
}

func (e *Entry) TextContent() string {
	return makePlainText(e.Content)
}

func (e *Entry) EnsureMaps() {
	if e.Properties == nil {
		e.Properties = map[string]interface{}{}
	}

	if e.Taxonomies == nil {
		e.Taxonomies = map[string][]string{}
	}
}

func (e *Entry) Update(newProps map[string][]interface{}) error {
	props := typed.New(mf2.Flatten(newProps))

	// Micropublish.net sends the file name that was uploaded through
	// the media endpoint as a property. This is unnecessary.
	delete(props, "file")

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

	if category, others := getCategoryStrings(props); len(category)+len(others) > 0 {
		if len(category) > 0 {
			// TODO: make 'tags' customizable.
			e.Taxonomies["tags"] = lo.Uniq(append(e.Taxonomy("tags"), category...))
		}

		if len(others) > 0 {
			e.Properties["category"] = others
		} else {
			delete(e.Properties, "category")
		}
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

func makePlainText(text string) string {
	text = htmlRemover.Sanitize(text)
	// Unescapes html entities.
	text = html.UnescapeString(text)
	text = stripMarkdown.Strip(text)
	return text
}

func getCategoryStrings(props typed.Typed) ([]string, []interface{}) {
	if v, ok := props.StringIf("category"); ok {
		return []string{v}, nil
	}

	// Slight modification of StringsIf so we capture
	// all string elements instead of blocking when there is none.
	// Tags can also be objects, such as tagged people as seen in
	// here: https://ownyourswarm.p3k.io/docs
	value, ok := props["category"]
	if !ok {
		return nil, nil
	}

	if tags, ok := value.([]string); ok {
		return tags, nil
	}

	if a, ok := value.([]interface{}); ok {
		tags := []string{}
		others := []interface{}{}

		for i := 0; i < len(a); i++ {
			if v, ok := a[i].(string); ok {
				tags = append(tags, v)
			} else {
				others = append(others, a[i])
			}
		}

		return tags, others
	}

	return nil, nil
}

type Entries []*Entry

func (ee Entries) AsLogs() Logs {
	l := Logs{}

	for _, e := range ee {
		mm := e.Helper()
		sub := mm.Sub(mm.TypeProperty())

		name := e.Title
		author := ""

		if sub != nil {
			if n := sub.Name(); n != "" {
				name = n
			}

			author = sub.String("author")
		}

		l = append(l, Log{
			Name:   name,
			Author: author,
			Rating: mm.Int("rating"),
			Date:   e.Published,
			URL:    e.ID,
		})
	}

	return l
}
