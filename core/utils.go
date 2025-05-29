package core

import (
	"html"
	"net/url"
	"path"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	stripMarkdown "github.com/writeas/go-strip-markdown/v2"
)

func cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	if id == "" {
		return "/"
	}
	return "/" + id + "/"
}

var htmlRemover = bluemonday.StrictPolicy()

func makePlainText(text string) string {
	text = normalizeNewlines(text)
	text = strings.Replace(text, "\n", " ", -1)
	text = htmlRemover.Sanitize(text)
	// Unescapes html entities.
	text = html.UnescapeString(text)
	text = stripMarkdown.Strip(text)

	return text
}

func normalizeNewlines(d string) string {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = strings.Replace(d, "\r\n", "\n", -1)
	// replace CF \r (mac) with LF \n (unix)
	d = strings.Replace(d, "\r", "\n", -1)
	return d
}

func truncateString(str string, length int) string {
	if length <= 0 {
		return ""
	}

	truncated := ""
	count := 0
	for _, char := range str {
		truncated += string(char)
		count++
		if count >= length {
			break
		}
	}
	return strings.TrimSpace(truncated)
}

func truncateStringWithEllipsis(str string, length int) string {
	str = strings.TrimSpace(str)
	newStr := truncateString(str, length)
	if newStr != str {
		newStr += "…"
	}

	return newStr
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	u2 := new(url.URL)
	*u2 = *u
	if u.User != nil {
		u2.User = new(url.Userinfo)
		*u2.User = *u.User
	}
	return u2
}
