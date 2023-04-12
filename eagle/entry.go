package eagle

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	stripMarkdown "github.com/writeas/go-strip-markdown"
	yaml "gopkg.in/yaml.v2"
)

type Entry struct {
	FrontMatter
	Path      string // The original path of the file. Might be empty.
	ID        string
	Permalink string
	Content   string
}

func (e *Entry) IsList() bool {
	return strings.Contains(e.Path, "_index.md")
}

func (e *Entry) Deleted() bool {
	if e.FrontMatter.ExpiryDate.IsZero() {
		return false
	}

	return e.FrontMatter.ExpiryDate.Before(time.Now())
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

var htmlRemover = bluemonday.StrictPolicy()

func makePlainText(text string) string {
	text = htmlRemover.Sanitize(text)
	// Unescapes html entities.
	text = html.UnescapeString(text)
	text = stripMarkdown.Strip(text)
	return text
}

type Entries []*Entry
