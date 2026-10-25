package server

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeRedirect(t *testing.T) {
	for redirect, expected := range map[string]string{
		"":                    "/",
		"/":                   "/",
		"/panel?x=1":          "/panel?x=1",
		"/panel/edit/content": "/panel/edit/content",
		"//evil.com":          "/",
		"///evil.com":         "/",
		`/\evil.com`:          "/",
		"https://evil.com":    "/",
		"javascript:alert(1)": "/",
		"/\t/evil.com":        "/",
		"evil.com":            "/",
	} {
		r := httptest.NewRequest("GET", "/panel/logout?redirect="+url.QueryEscape(redirect), nil)
		assert.Equal(t, expected, safeRedirect(r), redirect)
	}
}
