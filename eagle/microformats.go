package eagle

import (
	"errors"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/v2/pkg/mf2"
	"github.com/hacdias/eagle/v2/pkg/micropub"
	"github.com/karlseguin/typed"
)

var typeToSection = map[micropub.Type]string{
	micropub.TypeReply:    "micro",
	micropub.TypeNote:     "micro",
	micropub.TypeArticle:  "articles",
	micropub.TypeLike:     "likes",
	micropub.TypeRepost:   "reposts",
	micropub.TypeBookmark: "bookmarks",
	micropub.TypeCheckin:  "checkins",
}

func (e *Eagle) FromMicroformats(id string, data typed.Typed) (*Entry, error) {
	id = e.cleanID(id)
	permalink, err := e.makePermalink(id)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Frontmatter: Frontmatter{},
		ID:          id,
		Permalink:   permalink,
	}

	err = e.fromMicroformats(entry, data)
	return entry, err
}

func (e *Eagle) fromMicroformats(entry *Entry, data typed.Typed) error {
	postType := micropub.DiscoverType(data)
	switch postType {
	case micropub.TypeReply, micropub.TypeNote, micropub.TypeArticle,
		micropub.TypeLike, micropub.TypeRepost, micropub.TypeBookmark,
		micropub.TypeCheckin:
		if entry.Section == "" {
			entry.Section = typeToSection[postType]
		}
	default:
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

	if content, ok := data.StringsIf("content"); ok {
		content := joinString(content)
		entry.Content = content
		delete(data, "content")
	} else if _, ok := data["content"]; ok {
		return errors.New("could not parse content field")
	}

	if name, ok := data.StringsIf("name"); ok {
		entry.Title = joinString(name)
		delete(data, "name")
	}

	if summary, ok := data.StringsIf("summary"); ok {
		entry.Description = joinString(summary)
		delete(data, "summary")
	}

	if status, ok := data.StringsIf("post-status"); ok {
		it := joinString(status)
		if it == "draft" {
			entry.Draft = true
		}
		delete(data, "post-status")
	}

	if visibility, ok := data.StringsIf("visibility"); ok {
		it := joinString(visibility)
		if it == "private" {
			entry.Private = true
		}
		delete(data, "visibility")
	}

	if entry.Properties == nil {
		entry.Properties = map[string]interface{}{}
	}
	dd := interface{}(map[string]interface{}(data))

	entry.Properties = mf2.Flatten(interface{}(dd)).(map[string]interface{})
	return nil
}

func (e *Eagle) UpdateEntry(entry *Entry, data typed.Typed) error {
	return e.fromMicroformats(entry, data)
}

func (e *Eagle) ToMicroformats(entry *Entry) map[string]interface{} {
	properties := mf2.Deflatten(entry.Properties).(map[string]interface{})

	if !entry.Published.IsZero() {
		properties["published"] = []interface{}{entry.Published.Format(time.RFC3339)}
	}

	if !entry.Updated.IsZero() {
		properties["updated"] = []interface{}{entry.Updated.Format(time.RFC3339)}
	}

	properties["content"] = []interface{}{entry.Content}

	if entry.Title != "" {
		properties["name"] = []interface{}{entry.Title}
	}

	if entry.Description != "" {
		properties["summary"] = []interface{}{entry.Description}
	}

	if entry.Draft {
		properties["post-status"] = []interface{}{"draft"}
	}

	if entry.Private {
		properties["visibility"] = []interface{}{"private"}
	}

	return properties
}

func joinString(arr []string) string {
	return strings.TrimSpace(strings.Join(arr, "\n"))
}
