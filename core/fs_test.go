package core

import (
	"net/http"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRedirects(t *testing.T) {
	tests := []struct {
		title    string
		content  string
		expected map[string]Redirect
	}{
		{
			title:   "Two Fields",
			content: "/a /b\n",
			expected: map[string]Redirect{
				"/a": {To: "/b", Status: http.StatusMovedPermanently},
			},
		},
		{
			title:   "Three Fields",
			content: "/a /b 302\n/c /d 307\n/e /f 308\n/g /h 301\n",
			expected: map[string]Redirect{
				"/a": {To: "/b", Status: http.StatusFound},
				"/c": {To: "/d", Status: http.StatusTemporaryRedirect},
				"/e": {To: "/f", Status: http.StatusPermanentRedirect},
				"/g": {To: "/h", Status: http.StatusMovedPermanently},
			},
		},
		{
			title:   "Extra Whitespace",
			content: "  /a   /b\t302  \n\n/c  /d\r\n",
			expected: map[string]Redirect{
				"/a": {To: "/b", Status: http.StatusFound},
				"/c": {To: "/d", Status: http.StatusMovedPermanently},
			},
		},
		{
			title:   "Comments",
			content: "# Old blog\n/a /b\n  # 302 /c /d\n#/e /f\n",
			expected: map[string]Redirect{
				"/a": {To: "/b", Status: http.StatusMovedPermanently},
			},
		},
		{
			title:   "Fragment",
			content: "/contact /more/#contact 302\n",
			expected: map[string]Redirect{
				"/contact": {To: "/more/#contact", Status: http.StatusFound},
			},
		},
	}

	for _, tt := range tests {
		co := &Core{sourceFS: &afero.Afero{Fs: afero.NewMemMapFs()}}
		require.NoError(t, co.sourceFS.WriteFile(RedirectsFile, []byte(tt.content), 0644))

		redirects, err := co.GetRedirects(false)
		require.NoError(t, err, "failed for title: %s", tt.title)
		assert.Equal(t, tt.expected, redirects, "failed for title: %s", tt.title)
	}
}

func TestGetRedirectsMalformed(t *testing.T) {
	for _, line := range []string{
		"/a /b 200",
		"/a /b 304",
		"/a /b abc",
		"/a /b 302 extra",
		"/a",
	} {
		co := &Core{sourceFS: &afero.Afero{Fs: afero.NewMemMapFs()}}
		require.NoError(t, co.sourceFS.WriteFile(RedirectsFile, []byte(line+"\n/c /d\n"), 0644))

		_, err := co.GetRedirects(false)
		assert.Error(t, err, line)

		redirects, err := co.GetRedirects(true)
		require.NoError(t, err, line)
		assert.Equal(t, map[string]Redirect{
			"/c": {To: "/d", Status: http.StatusMovedPermanently},
		}, redirects, line)
	}
}
