package core

import (
	"fmt"
	"html"
	"path"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	stripMarkdown "github.com/writeas/go-strip-markdown"
)

func newTimeSlug(t time.Time) string {
	ns := t.Nanosecond()
	for ns > 99 {
		ns /= 10
	}
	return fmt.Sprintf("%02dh%02dm%02ds%02d", t.Hour(), t.Minute(), t.Second(), ns)
}

func NewID(slug string, t time.Time) string {
	if t.IsZero() {
		t = time.Now().Local()
	}

	if slug == "" {
		slug = newTimeSlug(t)
	}

	return fmt.Sprintf("/%04d/%02d/%02d/%s", t.Year(), t.Month(), t.Day(), slug)
}

func cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}

// Borrowed from https://github.com/jlelse/GoBlog/blob/master/utils.go
func Slugify(str string) string {
	return strings.Map(func(c rune) rune {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			// Is lower case ASCII or number, return unmodified
			return c
		} else if c >= 'A' && c <= 'Z' {
			// Is upper case ASCII, make lower case
			return c + 'a' - 'A'
		} else if c == ' ' || c == '-' || c == '_' {
			// Space, replace with '-'
			return '-'
		} else {
			// Drop character
			return -1
		}
	}, str)
}

var htmlRemover = bluemonday.StrictPolicy()

func makePlainText(text string) string {
	text = htmlRemover.Sanitize(text)
	// Unescapes html entities.
	text = html.UnescapeString(text)
	text = stripMarkdown.Strip(text)
	return text
}
