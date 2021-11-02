package eagle

import (
	"net"
	urlpkg "net/url"
)

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
