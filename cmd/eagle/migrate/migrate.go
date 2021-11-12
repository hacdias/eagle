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
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/log"
	"github.com/thoas/go-funk"
)

const dataPath = "testing/hacdias.com/data"
const oldContentPath = "testing/hacdias.com/content/"
const newContentPath = "testing/hacdias.com/content2/"

func Migrate() error {
	c, err := config.Parse()
	if err != nil {
		return err
	}

	defer func() {
		_ = log.L().Sync()
	}()

	e, err := eagle.NewEagle(c)
	if err != nil {
		return err
	}

	err = getReads(e)
	if err != nil {
		return err
	}

	entries, err := getAllEntries()
	if err != nil {
		return fmt.Errorf("error getting entries: %w", err)
	}

	aliases := ""
	sections := []string{}

	for _, oldEntry := range entries {
		if oldEntry.Metadata.Title == "Stream" {
			aliases += "/stream /\n"
			aliases += "/stream/feed.json /feed.json\n"
			aliases += "/stream/feed.xml /feed.atom\n"
			continue
		}

		if oldEntry.Section() != "" {
			sections = append(sections, oldEntry.Section())
		}

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

	_, err = copy(filepath.Join(dataPath, "blogroll.json"), filepath.Join(newContentPath, "links/_blogroll.json"))
	if err != nil {
		return err
	}
	_, err = copy(filepath.Join(dataPath, "music.json"), filepath.Join(newContentPath, "listens/_stats.json"))
	if err != nil {
		return err
	}
	_, err = copy(filepath.Join(dataPath, "watches.json"), filepath.Join(newContentPath, "watches/_stats.json"))
	if err != nil {
		return err
	}

	sections = funk.UniqString(sections)

	for _, section := range sections {
		aliases += fmt.Sprintf("/%s/feed.json /%s.json\n", section, section)
		aliases += fmt.Sprintf("/%s/feed.xml /%s.atom\n", section, section)
	}

	return saveAliases(aliases)
}

func convertEntry(oldEntry *Entry) *entry.Entry {
	newEntry := &entry.Entry{
		Frontmatter: entry.Frontmatter{
			Title:              oldEntry.Metadata.Title,
			Description:        oldEntry.Metadata.Description,
			Draft:              oldEntry.Metadata.Draft,
			Deleted:            !oldEntry.Metadata.ExpiryDate.IsZero(),
			Private:            false,
			NoShowInteractions: oldEntry.Metadata.NoMentions,
			Published:          oldEntry.Metadata.Date,
			Updated:            oldEntry.Metadata.Lastmod,
			Properties:         map[string]interface{}{},
		},
		Content: oldEntry.Content,
	}

	if oldEntry.Section() != "" && strings.Count(oldEntry.ID, "/") != 1 {
		newEntry.Sections = append(newEntry.Sections, oldEntry.Section())
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
		newEntry.Sections = append(newEntry.Sections, "replies")
	} else if oldEntry.Section() == "micro" {
		newEntry.Sections = append(newEntry.Sections, "notes")
	}

	if newEntry.Title == "Listens" {
		newEntry.Template = "listens"
	}

	if newEntry.Title == "Links" {
		newEntry.Template = "links"
	}

	if newEntry.Title == "Watches" {
		newEntry.Template = "watches"
	}

	if newEntry.Title == "Guestbook" {
		newEntry.Template = "guestbook"
	}

	if oldEntry.Section() == "photos" && oldEntry.Metadata.Photo != nil {
		photos := []interface{}{}

		for _, v := range oldEntry.Metadata.Photo {
			if s, ok := v.(string); ok {
				photos = append(photos, "cdn:/"+s)
			} else {
				m := v.(map[interface{}]interface{})
				value := m["value"].(string)

				photos = append(photos, map[string]interface{}{
					"value": "cdn:/" + value,
					"alt":   m["alt"],
				})
			}
		}

		newEntry.Properties["photo"] = photos
		newEntry.PhotoClass = oldEntry.Metadata.PhotoClass
	}

	return newEntry
}

func getAliases(oldEntry *Entry, newEntry *entry.Entry) (aliases string) {
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

func moveFiles(oldEntry *Entry, newEntry *entry.Entry) error {
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

func handleExternal(e *eagle.Eagle, oldEntry *Entry, newEntry *entry.Entry) error {
	var context map[string]interface{}

	if oldEntry.Metadata.ReplyTo != nil {
		context = convert(oldEntry.Metadata.ReplyTo)
	}

	// Targets and webmentions
	filename := filepath.Join(dataPath, "content", oldEntry.Metadata.DataID+".json")
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
