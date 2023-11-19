package core

import (
	"bytes"
	urlpkg "net/url"
	"path/filepath"

	"github.com/samber/lo"
	"willnorris.com/go/webmention"
)

// GetEntryLinks gets the links found in the HTML rendered version of the entry.
// This uses the latest available build to check for the links. Entry must have
// .h-entry and .e-content classes.
func (co *Core) GetEntryLinks(permalink string) ([]string, error) {
	url, err := urlpkg.Parse(permalink)
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(co.buildName, url.Path, "index.html")
	html, err := co.buildFS.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	targets, err := webmention.DiscoverLinksFromReader(bytes.NewBuffer(html), permalink, ".h-entry .e-content a, .h-entry .h-cite a")
	if err != nil {
		return nil, err
	}

	targets = (lo.Filter(targets, func(target string, _ int) bool {
		url, err := urlpkg.Parse(target)
		if err != nil {
			return false
		}

		return url.Scheme == "http" || url.Scheme == "https"
	}))

	return lo.Uniq(targets), nil
}

// IsLinkValid checks if the given link exists in the built version of the website.
func (co *Core) IsLinkValid(permalink string) (bool, error) {
	url, err := urlpkg.Parse(permalink)
	if err != nil {
		return false, err
	}

	_, err = co.buildFS.Stat(filepath.Join(co.buildName, url.Path))
	if err == nil {
		return true, nil
	}

	_, err = co.buildFS.Stat(filepath.Join(co.buildName, url.Path, "index.html"))
	if err == nil {
		return true, err
	}

	return false, nil
}
