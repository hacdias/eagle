package core

import (
	"html"
	"path"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	stripMarkdown "github.com/writeas/go-strip-markdown/v2"
)

func cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id + "/"
}

var htmlRemover = bluemonday.StrictPolicy()

func makePlainText(text string) string {
	text = htmlRemover.Sanitize(text)
	// Unescapes html entities.
	text = html.UnescapeString(text)
	text = stripMarkdown.Strip(text)
	text = normalizeNewlines(text)
	return text
}

func normalizeNewlines(d string) string {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = strings.Replace(d, "\r\n", "\n", -1)
	// replace CF \r (mac) with LF \n (unix)
	d = strings.Replace(d, "\r", "\n", -1)
	return d
}
