package xray

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var (
	htmlRemover    = bluemonday.StrictPolicy()
	spaceCollapser = regexp.MustCompile(`\s+`)
)

func SanitizeContent(data string) string {
	data = htmlRemover.Sanitize(data)
	data = strings.TrimSpace(data)
	data = spaceCollapser.ReplaceAllString(data, " ") // Collapse whitespaces
	return data
}
