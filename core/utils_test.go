package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakePlainText(t *testing.T) {
	tests := []struct {
		title    string
		input    string
		expected string
	}{
		{
			title:    "Plain Text",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			title:    "Heading With Attribute",
			input:    "\n\n## Hello, World! {#saturday}\n\n",
			expected: "Hello, World!",
		},
		{
			title:    "Hugo Shortcodes",
			input:    "\n\nHello, {{< favicon >}}World!\n\n",
			expected: "Hello, World!",
		},
		{
			title:    "Image With Attributes",
			input:    "\n![Image Alt](https://example.com)\n{width=1000}\n",
			expected: "Image Alt",
		},
		{
			title:    "More",
			input:    "\n<!--more-->\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, makePlainText(tt.input), "failed for title: %s", tt.title)
	}
}
