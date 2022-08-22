package util

import (
	"sort"
	"strings"
)

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

type GroupedByChars struct {
	Chars   []string
	Strings map[string][]string
}

func GroupByFirstChar(strs []string) *GroupedByChars {
	chars := []string{}
	byChar := map[string][]string{}

	for _, str := range strs {
		char := string([]rune(str)[0])

		_, ok := byChar[char]
		if !ok {
			chars = append(chars, char)
			byChar[char] = []string{}
		}

		byChar[char] = append(byChar[char], str)
	}

	sort.Strings(chars)

	for _, char := range chars {
		sort.Strings(byChar[char])
	}

	return &GroupedByChars{
		Chars:   chars,
		Strings: byChar,
	}
}
