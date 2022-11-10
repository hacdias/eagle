package util

import urlpkg "net/url"

func Domain(text string) string {
	u, err := urlpkg.Parse(text)
	if err != nil {
		return text
	}

	return u.Host
}
