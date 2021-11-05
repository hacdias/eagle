package eagle

import (
	"errors"
	"fmt"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v2/pkg/mf2"
	"github.com/karlseguin/typed"
	yaml "gopkg.in/yaml.v2"
)

type Entry struct {
	Frontmatter
	ID        string
	Permalink string
	Content   string

	summary string
}

func (e *Entry) MF2() *mf2.FlatHelper {
	if e.entry == nil {
		e.entry = mf2.NewFlatHelper(e.ToFlatMF2())
	}

	return e.entry
}

func (e *Entry) Tags() []string {
	m := typed.New(e.Properties)

	if v, ok := m.StringIf("category"); ok {
		return []string{v}
	}

	// Slight modification of StringsIf so we capture
	// all string elements instead of blocking when there is none.
	// Tags can also be objects, such as tagged people as seen in
	// here: https://ownyourswarm.p3k.io/docs
	value, ok := m["category"]
	if !ok {
		return []string{}
	}

	if n, ok := value.([]string); ok {
		return n
	}

	if a, ok := value.([]interface{}); ok {
		n := []string{}
		for i := 0; i < len(a); i++ {
			if v, ok := a[i].(string); ok {
				n = append(n, v)
			}
		}
		return n
	}

	return []string{}
}

func (e *Entry) Summary() string {
	if e.summary != "" {
		return e.summary
	}

	if strings.Contains(e.Content, "<!--more-->") {
		e.summary = strings.Split(e.Content, "<!--more-->")[0]
	} else if e.Description != "" {
		e.summary = e.Description
	} else {
		e.summary = truncate(e.Content, 300)
	}

	return e.summary
}

func (e *Entry) String() (string, error) {
	val, err := yaml.Marshal(&e.Frontmatter)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n\n%s\n", string(val), e.Content), nil
}

func (e *Entry) Templates() []string {
	tpls := []string{}
	if e.Section != "" {
		tpls = append(tpls, TemplateSingle+"."+e.Section)
	}
	tpls = append(tpls, TemplateSingle)
	return tpls
}

func (e *Entry) ContextURL() string {
	mf2 := e.MF2()
	urlStr := mf2.String(mf2.TypeProperty())
	return urlStr
}

// type Picture struct {
// 	Title string `yaml:"title,omitempty"`
// 	Slug  string `yaml:"slug,omitempty"`
// 	Hide  bool   `yaml:"hide,omitempty"`
// }

func (e *Eagle) GetEntry(id string) (*Entry, error) {
	id = e.cleanID(id)
	filepath, err := e.guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := e.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	entry, err := e.ParseEntry(id, string(raw))
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (e *Eagle) ParseEntry(id, raw string) (*Entry, error) {
	id = e.cleanID(id)
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	permalink, err := e.makePermalink(id)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		ID:          id,
		Permalink:   permalink,
		Content:     strings.TrimSpace(splits[1]),
		Frontmatter: Frontmatter{},
	}

	fr, err := unmarshalFrontmatter([]byte(splits[0]))
	if err != nil {
		return nil, err
	}

	entry.Frontmatter = *fr
	return entry, nil
}

func (e *Eagle) SaveEntry(entry *Entry) error {
	e.entriesMu.Lock()
	defer e.entriesMu.Unlock()

	entry.ID = e.cleanID(entry.ID)
	path, err := e.guessPath(entry.ID)
	if err != nil {
		if !os.IsNotExist(err) {
			return err

		}
		// Default path for new files is content/{slug}/index.md
		path = filepath.Join(ContentDirectory, entry.ID, "index.md")
	}

	err = e.SrcFs.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}

	str, err := entry.String()
	if err != nil {
		return err
	}

	err = e.Persist(path, []byte(str), "update "+entry.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	_ = e.IndexAdd(entry)
	return nil
}

func (e *Eagle) TransformEntry(id string, t func(*Entry) (*Entry, error)) (*Entry, error) {
	oldEntry, err := e.GetEntry(id)
	if err != nil {
		return nil, err
	}

	// TODO: make this open the file for writing and avoid using locks.

	newEntry, err := t(oldEntry)
	if err != nil {
		return nil, err
	}

	err = e.SaveEntry(newEntry)
	return newEntry, err
}

func (e *Eagle) GetAllEntries() ([]*Entry, error) {
	entries := []*Entry{}
	err := e.SrcFs.Walk(ContentDirectory, func(p string, info os.FileInfo, err error) error {
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

		entry, err := e.GetEntry(id)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}

func (e *Eagle) cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}

func (e *Eagle) guessPath(id string) (string, error) {
	path := filepath.Join(ContentDirectory, id, "index.md")
	_, err := e.SrcFs.Stat(path)
	if err == nil {
		return path, nil
	}

	return "", err
}

func (e *Eagle) makePermalink(id string) (string, error) {
	url, err := urlpkg.Parse(e.Config.Site.BaseURL) // Shouldn't this error be non-existent since we verify the BaseURL when parsing the conf?
	if err != nil {
		return "", err
	}
	url.Path = id
	return url.String(), nil
}

func (e *Eagle) PostSaveEntry(entry *Entry) {
	// 1. Check for context URL and fetch the data if needed.
	mm := entry.MF2()
	switch mm.PostType() {
	case mf2.TypeLike, mf2.TypeRepost, mf2.TypeReply:
		urlStr := entry.ContextURL()
		if urlStr != "" {
			err := e.fetchSaveContext(entry, urlStr)
			if err != nil {
				e.NotifyError(err)
			}
		}
	}

	// 2. Check if the post has a 'location' property and parse it
	// if it is a string.

	// 3. If it is a checkin, download map image.

	// TODO 4. Syndicate

	// 5. Webmentions
	err := e.SendWebmentions(entry)
	if err != nil {
		e.NotifyError(err)
	}
}

func (e *Eagle) fetchSaveContext(entry *Entry, urlStr string) error {
	sidecar, err := e.GetSidecar(entry)
	if err != nil {
		return fmt.Errorf("could not fetch sidecar for %s: %w", entry.ID, err)
	}

	if sidecar.Context != nil {
		return nil
	}

	context, err := e.getXRay(urlStr)
	if err != nil {
		return fmt.Errorf("could not fetch context xray for %s: %w", entry.ID, err)
	}

	return e.UpdateSidecar(entry, func(data *Sidecar) (*Sidecar, error) {
		data.Context = context
		return data, nil
	})
}
