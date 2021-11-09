package eagle

import (
	stripMarkdown "github.com/writeas/go-strip-markdown"
)

func sanitizePost(content string) string {
	content = stripMarkdown.Strip(content)
	return content
}
