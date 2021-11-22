package indieauth

import (
	"net"
	urlpkg "net/url"
	"strings"
)

// IsValidProfileURL validates the profile URL according to the specification.
// https://indieauth.spec.indieweb.org/#user-profile-url
func IsValidProfileURL(profile string) bool {
	url, err := urlpkg.Parse(profile)
	if err != nil {
		return false
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return false
	}

	if url.Path == "" {
		return false
	}

	if strings.Contains(url.Path, ".") || strings.Contains(url.Path, "..") {
		return false
	}

	if url.Fragment != "" {
		return false
	}

	if url.User.String() != "" {
		return false
	}

	if url.Port() != "" {
		return false
	}

	if net.ParseIP(profile) != nil {
		return false
	}

	return true
}

// IsValidClientIdentifier validates a client identifier according to the specification.
// https://indieauth.spec.indieweb.org/#client-identifier
func IsValidClientIdentifier(identifier string) bool {
	url, err := urlpkg.Parse(identifier)
	if err != nil {
		return false
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return false
	}

	if url.Path == "" {
		return false
	}

	if strings.Contains(url.Path, ".") || strings.Contains(url.Path, "..") {
		return false
	}

	if url.Fragment != "" {
		return false
	}

	if url.User.String() != "" {
		return false
	}

	if v := net.ParseIP(identifier); v != nil {
		return v.IsLoopback()
	}

	return true
}

// CanonicalizeURL checks if a URL has a path, and appends a path "/""
// if it has no path.
func CanonicalizeURL(urlStr string) string {
	// NOTE: parsing a URL without scheme will most likely put the host as path.
	// That's why I add it first.
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	if url.Path == "" {
		url.Path = "/"
	}

	return url.String()
}
