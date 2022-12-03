package xray

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var (
	htmlPolicy = bluemonday.StrictPolicy()
	spaces     = regexp.MustCompile(`\s+`)
	breaks     = regexp.MustCompile(`<br\s*/?>`)
)

func SanitizeContent(data string) string {
	data = breaks.ReplaceAllString(data, " ")
	data = htmlPolicy.Sanitize(data)
	data = strings.TrimSpace(data)
	// Collapse white spaces.
	data = spaces.ReplaceAllString(data, " ")
	return data
}
