package eagle

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/util"
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

var htmlRemover = bluemonday.StrictPolicy()

func makePlainText(text string) string {
	text = htmlRemover.Sanitize(text)
	// Unescapes html entities.
	text = html.UnescapeString(text)
	text = stripMarkdown.Strip(text)
	return text
}

type Entries []*Entry

func (ee Entries) AsLogs() Logs {
	logs := Logs{}

	for _, e := range ee {
		mm := e.Helper()
		sub := mm.Sub(mm.TypeProperty())

		l := Log{
			URL:    e.ID,
			Date:   e.Published,
			Rating: mm.Int("rating"),
		}

		if sub == nil {
			l.Name = e.Title
		} else {
			l.Name = sub.Name()
			l.Author = sub.String("author")
		}

		logs = append(logs, l)
	}

	return logs
}
