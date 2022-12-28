package renderer

import (
	urlpkg "net/url"
	"strings"
)

func (r *Renderer) ImageURL(urlStr string) string {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return ""
	}

	query := url.Query()
	query.Del("class")
	query.Del("id")
	query.Del("caption")
	url.RawQuery = query.Encode()

	if url.Scheme == "cdn" && r.m != nil {
		id := strings.TrimPrefix(url.Path, "/")
		return r.m.ImageURL(id)
	} else {
		return url.String()
	}
}
