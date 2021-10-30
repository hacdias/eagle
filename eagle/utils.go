package eagle

import (
	"net"
	urlpkg "net/url"
)

// From https://github.com/thoas/go-funk
func uniqString(a []string) []string {
	var (
		length  = len(a)
		seen    = make(map[string]struct{}, length)
		j       = 0
		results = make([]string, 0)
	)

	for i := 0; i < length; i++ {
		v := a[i]

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		results = append(results, v)
		j++
	}

	return results
}

func isPrivate(urlStr string) bool {
	url, _ := urlpkg.Parse(urlStr)
	if url == nil {
		return false
	}

	hostname := url.Hostname()
	if hostname == "localhost" {
		return true
	}

	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}

	return ip.IsPrivate() || ip.IsLoopback()
}
