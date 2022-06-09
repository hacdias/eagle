package util

import "strings"

func TruncateString(str string, length int) string {
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

func TruncateStringWithEllipsis(str string, length int) string {
	str = strings.TrimSpace(str)
	newStr := TruncateString(str, length)
	if newStr != str {
		newStr += "…"
	}

	return newStr
}
