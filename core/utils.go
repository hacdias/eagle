package core

import (
	"html"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
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
	text = htmlRemover.Sanitize(text)
	text = strings.ReplaceAll(text, moreSeparator, "")
	text = stripMarkdown(text)
	text = strings.ReplaceAll(text, "\n", " ")
	// Unescapes html entities.
	text = html.UnescapeString(text)
	return text
}

func normalizeNewlines(d string) string {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = strings.ReplaceAll(d, "\r\n", "\n")
	// replace CF \r (mac) with LF \n (unix)
	d = strings.ReplaceAll(d, "\r", "\n")
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

// Based on https://github.com/writeas/go-strip-markdown with some modifications.
var (
	listLeadersReg  = regexp.MustCompile(`(?m)^([\s\t]*)([\*\-\+]|\d\.)\s+`)
	headerReg       = regexp.MustCompile(`\n={2,}`)
	strikeReg       = regexp.MustCompile(`~~`)
	codeReg         = regexp.MustCompile("`{3}" + `.*\n`)
	htmlReg         = regexp.MustCompile("<(.*?)>")
	emphReg         = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	emphReg2        = regexp.MustCompile(`\*([^*]+)\*`)
	emphReg3        = regexp.MustCompile(`__([^_]+)__`)
	emphReg4        = regexp.MustCompile(`_([^_]+)_`)
	setextHeaderReg = regexp.MustCompile(`^[=\-]{2,}\s*$`)
	footnotesReg    = regexp.MustCompile(`\[\^.+?\](\: .*?$)?`)
	footnotes2Reg   = regexp.MustCompile(`\s{0,2}\[.*?\]: .*?$`)
	imagesReg       = regexp.MustCompile(`\!\[(.*?)\]\s?[\[\(].*?[\]\)]`)
	linksReg        = regexp.MustCompile(`\[(.*?)\][\[\(].*?[\]\)]`)
	blockquoteReg   = regexp.MustCompile(`>\s*`)
	refLinkReg      = regexp.MustCompile(`(?m)^\[(.*?)\]: (\S+)( ".*?")?\s*$`)
	atxHeaderReg    = regexp.MustCompile(`(?m)^\#{1,6}\s*([^#]+)\s*(\#{1,6})?$`)
	atxHeaderReg2   = regexp.MustCompile(`([\*_]{1,3})(\S.*?\S)?P1`)
	atxHeaderReg3   = regexp.MustCompile("(?m)(`{3,})" + `(.*?)?P1`)
	atxHeaderReg4   = regexp.MustCompile(`^-{3,}\s*$`)
	atxHeaderReg5   = regexp.MustCompile("`(.+?)`")
	atxHeaderReg6   = regexp.MustCompile(`\n{2,}`)
	shortcodesReg   = regexp.MustCompile(`(?m)^{.*}$`)
)

func stripMarkdown(s string) string {
	res := s
	res = shortcodesReg.ReplaceAllString(res, "")
	res = listLeadersReg.ReplaceAllString(res, "$1")
	res = headerReg.ReplaceAllString(res, "\n")
	res = strikeReg.ReplaceAllString(res, "")
	res = codeReg.ReplaceAllString(res, "")
	res = emphReg.ReplaceAllString(res, "$1")
	res = emphReg2.ReplaceAllString(res, "$1")
	res = emphReg3.ReplaceAllString(res, "$1")
	res = emphReg4.ReplaceAllString(res, "$1")
	res = htmlReg.ReplaceAllString(res, "$1")
	res = setextHeaderReg.ReplaceAllString(res, "")
	res = footnotesReg.ReplaceAllString(res, "")
	res = footnotes2Reg.ReplaceAllString(res, "")
	res = imagesReg.ReplaceAllString(res, "$1")
	res = linksReg.ReplaceAllString(res, "$1")
	res = blockquoteReg.ReplaceAllString(res, "  ")
	res = refLinkReg.ReplaceAllString(res, "")
	res = atxHeaderReg.ReplaceAllString(res, "$1")
	res = atxHeaderReg2.ReplaceAllString(res, "$2")
	res = atxHeaderReg3.ReplaceAllString(res, "$2")
	res = atxHeaderReg4.ReplaceAllString(res, "")
	res = atxHeaderReg5.ReplaceAllString(res, "$1")
	res = atxHeaderReg6.ReplaceAllString(res, "\n\n")
	return res
}
