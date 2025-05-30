package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntryStatus(t *testing.T) {
	tests := []struct {
		title          string
		content        string
		forcePermalink bool
		expected       string
	}{
		{
			title:          "Lorem Ipsum A",
			content:        "Breprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
			forcePermalink: false,
			expected:       "Lorem Ipsum A https://example.com/test-entry",
		},
		{
			title:          "Lorem Ipsum B",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
			forcePermalink: false,
			expected:       "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
		},
		{
			title:          "Lorem Ipsum C",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
			forcePermalink: false,
			expected:       "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
		},
		{
			title:          "Lorem Ipsum D",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
			forcePermalink: true,
			expected:       "Lorem Ipsum D https://example.com/test-entry",
		},
		{
			title:          "Lorem Ipsum E",
			content:        "",
			forcePermalink: false,
			expected:       "Lorem Ipsum E",
		},
		{
			title:          "Lorem Ipsum F",
			content:        "",
			forcePermalink: true,
			expected:       "Lorem Ipsum F https://example.com/test-entry",
		},
		{
			title:          "Lorem Ipsum G",
			content:        "commodo veniam est consectetur proident ipsum dolore fugiat duis voluptate",
			forcePermalink: false,
			expected:       "commodo veniam est consectetur proident ipsum dolore fugiat duis voluptate",
		},
		{
			title:          "Lorem Ipsum H",
			content:        "commodo veniam est consectetur proident ipsum dolore fugiat duis voluptate",
			forcePermalink: true,
			expected:       "commodo veniam est consectetur proident ipsum dolore fugiat duis voluptate https://example.com/test-entry",
		},
	}

	for _, tt := range tests {
		e := &Entry{
			FrontMatter: FrontMatter{
				Title: tt.title,
			},
			Content:   tt.content,
			Permalink: "https://example.com/test-entry",
		}

		status := e.Status(300, tt.forcePermalink)
		assert.Equal(t, tt.expected, status, "failed for title: %s", tt.title)
	}
}
