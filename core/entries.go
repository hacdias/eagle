package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const moreSeparator = "<!--more-->"

type FrontMatter struct {
	Title       string         `yaml:"title,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Draft       bool           `yaml:"draft,omitempty"`
	Date        time.Time      `yaml:"date,omitempty"`
	Lastmod     time.Time      `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time      `yaml:"expiryDate,omitempty"`
	NoIndex     bool           `yaml:"noIndex,omitempty"`
	Categories  []string       `yaml:"categories,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	Other       map[string]any `yaml:",inline"`
}

type Entry struct {
	FrontMatter
	ID        string
	IsList    bool
	Permalink string
	Content   string
}

func (e *Entry) Deleted() bool {
	if e.FrontMatter.ExpiryDate.IsZero() {
		return false
	}

	return e.FrontMatter.ExpiryDate.Before(time.Now())
}

func (e *Entry) Summary() string {
	if strings.Contains(e.Content, moreSeparator) {
		firstPart := strings.Split(e.Content, moreSeparator)[0]
		return strings.TrimSpace(makePlainText(firstPart))
	} else if content := e.TextContent(); content != "" {
		return truncateStringWithEllipsis(content, 300)
	} else {
		return content
	}
}

func (e *Entry) String() (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	err := enc.Encode(&e.FrontMatter)
	if err != nil {
		return "", err
	}

	text := fmt.Sprintf("---\n%s---\n\n%s\n", buf.String(), strings.TrimSpace(e.Content))
	text = strings.TrimSpace(text) + "\n"
	return normalizeNewlines(text), nil
}

func (e *Entry) TextContent() string {
	return makePlainText(e.Content)
}

type Entries []*Entry

// errIgnoredEntry is a locally used error to indicate this an errIgnoredEntry.
var errIgnoredEntry error = errors.New("ignored entry")

func (co *Core) GetEntry(id string) (*Entry, error) {
	// TODO: ideally this wouldn't be needed in the future if Eagle uses its
	// own templates. Or if there's a better way of printing this template.
	if id == "/_eagle/" {
		return nil, errIgnoredEntry
	}

	filename := co.entryFilenameFromID(id)
	raw, err := co.sourceFS.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	e, err := co.parseEntry(id, string(raw))
	if err != nil {
		return nil, err
	}

	// Ignore entries that are not built. This is a very simplified way and won't
	// really work for cascading builds.
	if v, ok := e.Other["_build"]; ok {
		if m, ok := v.(map[string]any); ok {
			if m["render"] == "never" {
				return nil, errIgnoredEntry
			}
		}
	}

	// TODO: make this configurable.
	e.IsList = strings.HasPrefix(id, "/categories/") ||
		strings.HasPrefix(id, "/tags/")

	return e, nil
}

func (co *Core) GetEntries(includeList bool) (Entries, error) {
	ee := Entries{}
	err := co.sourceFS.Walk(ContentDirectory, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(p, ".md") {
			return nil
		}

		id := strings.TrimPrefix(p, ContentDirectory)
		id = strings.TrimSuffix(id, ".md")
		id = strings.TrimSuffix(id, "_index")
		id = strings.TrimSuffix(id, "index")

		e, err := co.GetEntry(id)
		if err != nil {
			if errors.Is(err, errIgnoredEntry) {
				return nil
			}
			return err
		}

		if !e.IsList || includeList {
			ee = append(ee, e)
		}

		return nil
	})

	return ee, err
}

func (co *Core) SaveEntry(e *Entry) error {
	filename := co.entryFilenameFromID(e.ID)
	err := co.sourceFS.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return err
	}

	str, err := e.String()
	if err != nil {
		return err
	}

	err = co.WriteFile(filename, []byte(str), "entry: update "+e.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	return nil
}

func (co *Core) parseEntry(id, raw string) (*Entry, error) {
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	fr := &FrontMatter{}
	err := yaml.Unmarshal([]byte(splits[0]), &fr)
	if err != nil {
		return nil, err
	}

	id = cleanID(id)

	permalink := co.entryPermalinkFromID(id, fr)

	content := strings.TrimSpace(splits[1])
	if content != "" {
		// Fixes issue where goldmark is adding a <blockquote>
		// if the document ends with an HTML tag.
		content += "\n"
	}

	e := &Entry{
		ID:          id,
		Permalink:   permalink,
		Content:     content,
		FrontMatter: *fr,
	}

	return e, nil
}

func (f *Core) entryFilenameFromID(id string) string {
	path := filepath.Join(ContentDirectory, id, "_index.md")
	if _, err := f.sourceFS.Stat(path); err == nil {
		return path
	}

	return filepath.Join(ContentDirectory, id, "index.md")
}

// TODO: do not hardcore this. Instead, use Hugo's configuration to deduce
// and "back-engineer" how the permalinks are constructed. Then this can be used
// only in the parser code.
const SpecialSection = "posts"

func (co *Core) entryPermalinkFromID(id string, fr *FrontMatter) string {
	url := co.BaseURL()

	// TODO: very specific code.
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		url.Path = id
	} else if parts[1] == SpecialSection && !fr.Date.IsZero() {
		url.Path = fmt.Sprintf("/%04d/%02d/%02d/%s/", fr.Date.Year(), fr.Date.Month(), fr.Date.Day(), parts[len(parts)-2])
	} else if parts[1] == "categories" {
		url.Path = "/" + parts[2] + "/"
	} else {
		url.Path = id
	}

	return url.String()
}
