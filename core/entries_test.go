package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntryStatus(t *testing.T) {
	tests := []struct {
		title          string
		content        string
		maximumPosts   int
		forcePermalink bool
		expected       []string
	}{
		{
			title:          "Lorem Ipsum A",
			content:        "Breprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
			forcePermalink: false,
			maximumPosts:   1,
			expected:       []string{"Lorem Ipsum A https://example.com/test-entry"},
		},
		{
			title:          "Lorem Ipsum B",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
			forcePermalink: false,
			maximumPosts:   1,
			expected:       []string{"reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup"},
		},
		{
			title:          "Lorem Ipsum C",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
			forcePermalink: false,
			maximumPosts:   1,
			expected:       []string{"reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup"},
		},
		{
			title:          "Lorem Ipsum D",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
			forcePermalink: true,
			maximumPosts:   1,
			expected:       []string{"Lorem Ipsum D https://example.com/test-entry"},
		},
		{
			title:          "Lorem Ipsum E",
			content:        "",
			forcePermalink: false,
			maximumPosts:   1,
			expected:       []string{"Lorem Ipsum E"},
		},
		{
			title:          "Lorem Ipsum F",
			content:        "",
			forcePermalink: true,
			maximumPosts:   1,
			expected:       []string{"Lorem Ipsum F https://example.com/test-entry"},
		},
		{
			title:          "Lorem Ipsum G",
			content:        "commodo veniam est consectetur proident ipsum dolore fugiat duis voluptate",
			forcePermalink: false,
			maximumPosts:   1,
			expected:       []string{"commodo veniam est consectetur proident ipsum dolore fugiat duis voluptate"},
		},
		{
			title:          "Lorem Ipsum H",
			content:        "commodo veniam est consectetur proident ipsum dolore fugiat duis voluptate",
			forcePermalink: true,
			maximumPosts:   1,
			expected:       []string{"commodo veniam est consectetur proident ipsum dolore fugiat duis voluptate https://example.com/test-entry"},
		},
		{
			title:          "Lorem Ipsum F",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim",
			forcePermalink: false,
			maximumPosts:   2,
			expected: []string{
				"reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
				"reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim",
			},
		},
		{
			title:          "Lorem Ipsum G",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim",
			forcePermalink: true,
			maximumPosts:   2,
			expected:       []string{"Lorem Ipsum G https://example.com/test-entry"},
		},
		{
			title:          "Lorem Ipsum H",
			content:        "reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim eaa",
			forcePermalink: true,
			maximumPosts:   2,
			expected: []string{
				"reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim ea cillum ea sunt quis dolore enim cup",
				"reprehenderit velit nisi proident dolor commodo ipsum duis Lorem non voluptate est nostrud ipsum incididunt amet et ullamco enim deserunt velit amet est dolore ex enim pariatur id est proident proident reprehenderit elit ea Lorem incididunt officia laborum anim eaa https://example.com/test-entry",
			},
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

		statuses := e.Statuses(300, tt.maximumPosts, tt.forcePermalink)
		assert.Equal(t, tt.expected, statuses, "failed for title: %s", tt.title)
	}
}

func TestEntryIsPost(t *testing.T) {
	assert.True(t, (&Entry{
		FrontMatter: FrontMatter{},
		ID:          "/posts/2026/01/01/test-entry/",
	}).IsPost())
	assert.False(t, (&Entry{
		FrontMatter: FrontMatter{},
		ID:          "/about/",
	}).IsPost())
}
