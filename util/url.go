package util

import urlpkg "net/url"

func Domain(text string) string {
	u, err := urlpkg.Parse(text)
	if err != nil {
		return text
	}

	return u.Host
}

func StripScheme(url string) string {
	u, err := urlpkg.Parse(url)
	if err != nil {
		return url
	}

	return u.Host + u.Path
}
