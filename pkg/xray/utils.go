package xray

import (
	"regexp"
	"strings"
)

var (
	spaceCollapser = regexp.MustCompile(`\s+`)
)

func SanitizeContent(data string) string {
	data = strings.TrimSpace(data)
	data = spaceCollapser.ReplaceAllString(data, " ") // Collapse whitespaces
	return data
}
