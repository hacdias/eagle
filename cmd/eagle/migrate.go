package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/logging"
	"github.com/hacdias/eagle/v2/pkg/yaml"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate from the Hugo based website",
	// Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
			return err
		}

		aliases := ""

		for _, entry := range entries {
			newEntry := &eagle.Entry{
				Frontmatter: eagle.Frontmatter{
					Title:          entry.Metadata.Title,
					Description:    entry.Metadata.Description,
					Draft:          entry.Metadata.Draft,
					Deleted:        !entry.Metadata.ExpiryDate.IsZero(),
					Private:        false,
					NoInteractions: entry.Metadata.NoMentions,
					Emoji:          entry.Metadata.Emoji,
					Published:      entry.Metadata.Date,
					Updated:        entry.Metadata.Lastmod,
					Section:        entry.Section(),
					Properties:     map[string]interface{}{}, // TODO: fill this
				},
				ID:      entry.ID, // TODO:make new id
				Content: entry.Content,
			}

			var id string

			if newEntry.Published.IsZero() || strings.Count(entry.ID, "/") == 1 {
				id = newEntry.ID
			} else {
				year := newEntry.Published.Year()
				month := newEntry.Published.Month()
				day := newEntry.Published.Day()

				id = fmt.Sprintf("%04d/%02d/%02d/%s", year, month, day, entry.Slug())
				aliases += entry.ID + " " + id + "\n"
			}

			if entry.Metadata.Tags != nil && len(entry.Metadata.Tags) > 0 {
				newEntry.Properties["category"] = entry.Metadata.Tags
			}

			if entry.Metadata.Syndication != nil && len(entry.Metadata.Syndication) > 0 {
				newEntry.Properties["syndication"] = entry.Metadata.Syndication
			}

			if entry.Metadata.Aliases != nil {
				for _, alias := range entry.Metadata.Aliases {
					aliases += alias + " " + id + "\n"
				}
			}

			if entry.Metadata.ReplyTo != nil {
				newEntry.Properties["in-reply-to"] = entry.Metadata.ReplyTo.URL
				newEntry.Section = "replies"

				_, err = e.GetXRay(entry.Metadata.ReplyTo.URL)
				if err != nil {
					fmt.Println(err)
				}
			}

			if newEntry.Section == "micro" {
				newEntry.Section = "notes"
			}

			// Layout      string      `yaml:"layout,omitempty"`
			// Cover       *eagle.Picture   `yaml:"cover,omitempty"`

			newEntry.ID = id

			if false {
				err = e.SaveEntry(newEntry)
				if err != nil {
					return err
				}

				// Moving files
				dir := filepath.Dir(entry.Path)
				files, err := os.ReadDir(dir)
				if err != nil {
					return err
				}
				for _, file := range files {
					if file.Name() == "index.md" || file.Name() == "_index.md" || file.IsDir() {
						continue
					}

					_, err = copy(filepath.Join(dir, file.Name()), filepath.Join(otherPath, newEntry.ID, file.Name()))
					if err != nil {
						return err
					}
				}
			}

			// Targets and webmentions
			filename := filepath.Join(dataPath, entry.Metadata.DataID+".json")
			if _, err := os.Stat(filename); err == nil {
				data, err := os.ReadFile(filename)
				if err != nil {
					return err
				}

				var ed EntryData
				err = json.Unmarshal(data, &ed)
				if err != nil {
					return err
				}

				for _, wm := range ed.Webmentions {
					_, err = e.GetXRay(wm.URL)
					if err != nil {
						fmt.Println(err)
					}
				}

				err = e.TransformEntryData(newEntry, func(ned *eagle.EntryData) (*eagle.EntryData, error) {
					ned.Targets = ed.Targets
					ned.Webmentions = []string{}
					for _, wm := range ed.Webmentions {
						ned.Webmentions = append(ned.Webmentions, wm.URL)
					}

					return ned, nil
				})
				if err != nil {
					return err
				}

			}

			// fmt.Println(id)
		}

		// fmt.Println(aliases)

		return nil
	},
}

type Entry struct {
	Path     string
	ID       string
	Content  string
	Metadata Metadata
}

type Webmention struct {
	XRay `yaml:",inline"`
	// Specifically for webmentions received from https://webmention.io
	// TODO: remove this and compare webmentions via URL.
	WmID    int  `yaml:"wm-id,omitempty" json:"wm-id,omitempty"`
	Private bool `json:"private,omitempty"`
}
type EntryData struct {
	Targets     []string      `json:"targets"`
	Webmentions []*Webmention `json:"webmentions"`
}

// XRay is an xray of an external post. This is the format used to store
// Webmentions and ReplyTo context.
type XRay struct {
	Type    string    `yaml:"type,omitempty" json:"type,omitempty"`
	URL     string    `yaml:"url,omitempty" json:"url,omitempty"`
	Name    string    `yaml:"name,omitempty" json:"name,omitempty"`
	Content string    `yaml:"content,omitempty" json:"content,omitempty"`
	Date    time.Time `yaml:"date,omitempty" json:"date,omitempty"`
	Author  *Author   `yaml:"author,omitempty" json:"author,omitempty"`
}

type Author struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
	Photo string `yaml:"photo,omitempty" json:"photo,omitempty"`
}

type Metadata struct {
	DataID      string    `yaml:"dataId,omitempty"`
	Title       string    `yaml:"title,omitempty"`
	Description string    `yaml:"description,omitempty"`
	Tags        []string  `yaml:"tags,omitempty"`
	Date        time.Time `yaml:"date,omitempty"`
	Lastmod     time.Time `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time `yaml:"expiryDate,omitempty"`
	Syndication []string  `yaml:"syndication,omitempty"`
	ReplyTo     *XRay     `yaml:"replyTo,omitempty"`
	URL         string    `yaml:"url,omitempty"`
	Aliases     []string  `yaml:"aliases,omitempty"`
	Emoji       string    `yaml:"emoji,omitempty"`
	Layout      string    `yaml:"layout,omitempty"`
	NoMentions  bool      `yaml:"noMentions,omitempty"`
	Cover       *Picture  `yaml:"cover,omitempty"`
	Draft       bool      `yaml:"draft,omitempty"`
	Growth      string    `yaml:"growth,omitempty"`
}

type Picture struct {
	Slug string
}

func (e *Entry) Section() string {
	cleanID := strings.TrimPrefix(e.ID, "/")
	cleanID = strings.TrimSuffix(cleanID, "/")

	section := ""
	if strings.Count(cleanID, "/") >= 1 {
		section = strings.Split(cleanID, "/")[0]
	}
	return section
}

func (e *Entry) Slug() string {
	cleanID := strings.TrimPrefix(e.ID, "/")
	cleanID = strings.TrimSuffix(cleanID, "/")
	a := strings.Split(cleanID, "/")
	return a[len(a)-1]
}

const dataPath = "testing/hacdias.com/data/content"
const basePath = "testing/hacdias.com/content/"
const otherPath = "testing/hacdias.com/content2/"

func getAllEntries() ([]*Entry, error) {
	entries := []*Entry{}
	err := filepath.Walk(basePath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(p, ".md") {
			return nil
		}

		id := strings.TrimPrefix(p, basePath)
		id = strings.TrimSuffix(id, ".md")
		id = strings.TrimSuffix(id, "_index")
		id = strings.TrimSuffix(id, "index")

		entry, err := getEntry(id)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}

func getEntry(id string) (*Entry, error) {
	filepath, err := guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	entry, err := parseEntry(id, string(raw))
	if err != nil {
		return nil, err
	}

	entry.Path = filepath
	return entry, nil
}

func parseEntry(id, raw string) (*Entry, error) {
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	entry := &Entry{
		ID:       id,
		Content:  strings.TrimSpace(splits[1]),
		Metadata: Metadata{},
	}

	err := yaml.Unmarshal([]byte(splits[0]), &entry.Metadata)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func guessPath(id string) (string, error) {
	path := filepath.Join(basePath, id, "index.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join(basePath, id, "_index.md")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else {
		return "", err
	}
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
