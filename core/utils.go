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

func newID(section, slug string) string {
	return fmt.Sprintf("/%s/%04d/%s/", section, time.Now().Year(), slug)
}

func cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id + "/"
}

// Borrowed from https://github.com/jlelse/GoBlog/blob/master/utils.go
func slugify(str string) string {
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
