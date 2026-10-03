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

func TestGetEntryLinks(t *testing.T) {
	tests := []struct {
		name             string
		entry            *Entry
		withSyndications bool
		expected         []string
	}{
		{
			name:     "inline link with entity reference",
			entry:    &Entry{Content: "[a](https://example.com/a?x=1&amp;y=2)"},
			expected: []string{"https://example.com/a?x=1&y=2"},
		},
		{
			name:     "autolink and linkify",
			entry:    &Entry{Content: "<https://example.com/a> and https://example.com/b?z=1"},
			expected: []string{"https://example.com/a", "https://example.com/b?z=1"},
		},
		{
			name:     "escapes, spaces and non-ASCII",
			entry:    &Entry{Content: "[a](https://example.com/a\\_b) [b](<https://example.com/a b>) [c](https://ü.com/é)"},
			expected: []string{"https://example.com/a_b", "https://example.com/a%20b", "https://%C3%BC.com/%C3%A9"},
		},
		{
			name:     "non-HTTP links are filtered",
			entry:    &Entry{Content: "<me@example.com> [a](ftp://example.com/a)"},
			expected: []string{},
		},
		{
			name: "bookmark, syndications and duplicates",
			entry: &Entry{
				FrontMatter: FrontMatter{
					Syndications: []string{"https://example.com/syndication"},
					Other:        map[string]any{"bookmark-of": "https://example.com/bookmark"},
				},
				Content: "[a](https://example.com/bookmark) [b](https://example.com/a)",
			},
			withSyndications: true,
			expected:         []string{"https://example.com/bookmark", "https://example.com/syndication", "https://example.com/a"},
		},
		{
			name: "syndications excluded",
			entry: &Entry{
				FrontMatter: FrontMatter{Syndications: []string{"https://example.com/syndication"}},
				Content:     "[a](https://example.com/a)",
			},
			expected: []string{"https://example.com/a"},
		},
	}

	co := &Core{}
	for _, tt := range tests {
		links, err := co.GetEntryLinks(tt.entry, tt.withSyndications)
		assert.NoError(t, err, "failed for: %s", tt.name)
		assert.Equal(t, tt.expected, links, "failed for: %s", tt.name)
	}
}
