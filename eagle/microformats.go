package eagle

import (
	"errors"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/v2/pkg/jf2"
	"github.com/karlseguin/typed"
	"github.com/thoas/go-funk"
)

func (e *Eagle) FromMicroformats(id string, mf2Data map[string][]interface{}) (*Entry, error) {
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

	err = e.fromMicroformats(entry, mf2Data)
	return entry, err
}

func (e *Eagle) UpdateEntry(entry *Entry, mf2Data map[string][]interface{}) error {
	return e.fromMicroformats(entry, mf2Data)
}

func (e *Eagle) fromMicroformats(entry *Entry, mf2Data map[string][]interface{}) error {
	data := typed.New(jf2.FromMicroformats(mf2Data))

	postType, _ := jf2.DiscoverType(data)

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

func (entry *Entry) ToMicroformats() map[string][]interface{} {
	properties := jf2.ToMicroformats(entry.Properties)

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
	} else {
		properties["post-status"] = []interface{}{"published"}
	}

	if entry.Private {
		properties["visibility"] = []interface{}{"private"}
	} else {
		properties["visibility"] = []interface{}{"public"}
	}

	return properties
}
