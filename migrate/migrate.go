package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/logging"
)

const dataPath = "testing/hacdias.com/data/content"
const oldContentPath = "testing/hacdias.com/content/"
const newContentPath = "testing/hacdias.com/content2/"

func Migrate() error {
	c, err := config.Parse()
	if err != nil {
		return err
	}

	defer func() {
		_ = logging.L().Sync()
	}()

	e, err := eagle.NewEagle(c)
	if err != nil {
		return err
	}

	entries, err := getAllEntries()
	if err != nil {
		return fmt.Errorf("error getting entries: %w", err)
	}

	aliases := ""

	for _, oldEntry := range entries {
		newEntry := convertEntry(oldEntry)
		aliases += getAliases(oldEntry, newEntry)

		err = e.SaveEntry(newEntry)
		if err != nil {
			return err
		}

		err = moveFiles(oldEntry, newEntry)
		if err != nil {
			return err
		}

		err = handleExternal(e, oldEntry, newEntry)
		if err != nil {
			return err
		}
	}

	return saveAliases(aliases)
}

func convertEntry(oldEntry *Entry) *eagle.Entry {
	newEntry := &eagle.Entry{
		Frontmatter: eagle.Frontmatter{
			Title:          oldEntry.Metadata.Title,
			Description:    oldEntry.Metadata.Description,
			Draft:          oldEntry.Metadata.Draft,
			Deleted:        !oldEntry.Metadata.ExpiryDate.IsZero(),
			Private:        false,
			NoInteractions: oldEntry.Metadata.NoMentions,
			Emoji:          oldEntry.Metadata.Emoji,
			Published:      oldEntry.Metadata.Date,
			Updated:        oldEntry.Metadata.Lastmod,
			Section:        oldEntry.Section(),
			Properties:     map[string]interface{}{},
		},
		Content: oldEntry.Content,
	}

	if newEntry.Published.IsZero() || strings.Count(oldEntry.ID, "/") == 1 {
		newEntry.ID = oldEntry.ID
	} else {
		year := newEntry.Published.Year()
		month := newEntry.Published.Month()
		day := newEntry.Published.Day()

		newEntry.ID = fmt.Sprintf("/%04d/%02d/%02d/%s", year, month, day, oldEntry.Slug())
	}

	if oldEntry.Metadata.Tags != nil && len(oldEntry.Metadata.Tags) > 0 {
		newEntry.Properties["category"] = oldEntry.Metadata.Tags
	}

	if oldEntry.Metadata.Syndication != nil && len(oldEntry.Metadata.Syndication) > 0 {
		newEntry.Properties["syndication"] = oldEntry.Metadata.Syndication
	}

	if oldEntry.Metadata.ReplyTo != nil {
		newEntry.Properties["in-reply-to"] = oldEntry.Metadata.ReplyTo.URL
		newEntry.Section = "replies"
	}

	if newEntry.Section == "micro" {
		newEntry.Section = "notes"
	}

	// TODO: deal with cover image.
	return newEntry
}

func getAliases(oldEntry *Entry, newEntry *eagle.Entry) (aliases string) {
	if oldEntry.ID != newEntry.ID {
		aliases += oldEntry.ID + " " + newEntry.ID + "\n"
	}

	if oldEntry.Metadata.Aliases != nil {
		for _, alias := range oldEntry.Metadata.Aliases {
			aliases += alias + " " + newEntry.ID + "\n"
		}
	}

	return aliases
}

func moveFiles(oldEntry *Entry, newEntry *eagle.Entry) error {
	dir := filepath.Dir(oldEntry.Path)
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if file.Name() == "index.md" || file.Name() == "_index.md" {
			continue
		}

		_, err = copy(filepath.Join(dir, file.Name()), filepath.Join(newContentPath, newEntry.ID, file.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

func convert(x *XRay) map[string]interface{} {
	r := map[string]interface{}{
		"type": "entry",
	}

	if x.URL != "" {
		r["url"] = x.URL
	}

	if x.Content != "" {
		r["content"] = x.Content
	}

	if !x.Date.IsZero() {
		r["published"] = x.Date.Format(time.RFC3339)
	}

	if x.Type != "" {
		r["post-type"] = x.Type
	}

	if x.Author != nil {
		r["author"] = map[string]interface{}{
			"name":  x.Author.Name,
			"type":  "card",
			"url":   x.Author.URL,
			"photo": x.Author.Photo,
		}
	}

	return r
}

func handleExternal(e *eagle.Eagle, oldEntry *Entry, newEntry *eagle.Entry) error {
	var context map[string]interface{}

	if oldEntry.Metadata.ReplyTo != nil {
		context = convert(oldEntry.Metadata.ReplyTo)
	}

	// Targets and webmentions
	filename := filepath.Join(dataPath, oldEntry.Metadata.DataID+".json")
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		var ed EntryData
		err = json.Unmarshal(data, &ed)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}

		webmentions := []map[string]interface{}{}

		for _, wm := range ed.Webmentions {
			w := convert(&wm.XRay)
			if wm.WmID != 0 {
				w["wm-id"] = w
			}

			webmentions = append(webmentions, convert(&wm.XRay))
		}

		err = e.UpdateSidecar(newEntry, func(ned *eagle.Sidecar) (*eagle.Sidecar, error) {
			ned.Targets = ed.Targets
			ned.Webmentions = webmentions
			ned.Context = context
			return ned, nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func saveAliases(aliases string) error {
	return os.WriteFile(filepath.Join(filepath.Dir(newContentPath), "aliases"), []byte(aliases), 0644)
}
