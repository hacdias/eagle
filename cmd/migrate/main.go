package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/services"
	"github.com/hacdias/eagle/yaml"
)

func main() {
	moveWebmentions()
}

func moveWebmentions() {
	c, err := config.Parse()
	if err != nil {
		panic(err)
	}

	eagle, err := services.NewEagle(c)
	if err != nil {
		panic(err)
	}

	entries, err := eagle.GetAll()
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(filepath.Join(c.Hugo.Source, "data", "interactions"), 0777)
	if err != nil {
		panic(err)
	}

	fmt.Printf("got %d entries\n", len(entries))

	for _, entry := range entries {
		wmpath := filepath.Dir(entry.Path)
		wmpath = filepath.Join(wmpath, "mentions.json")

		if _, err := os.Stat(wmpath); err == nil {
			raw, err := ioutil.ReadFile(wmpath)
			if err != nil {
				panic(err)
			}

			var mentions []services.StoredWebmention
			err = json.Unmarshal(raw, &mentions)
			if err != nil {
				panic(err)
			}

			newMentions := []services.EmbeddedEntry{}
			for _, m := range mentions {
				date, err := dateparse.ParseStrict(m.Date)
				if err != nil {
					panic(err)
				}
				nm := services.EmbeddedEntry{
					WmID: uint(m.ID),
					Type: m.Type,
					URL:  m.URL,
					Date: date,
				}

				if m.Content.Text != "" {
					nm.Content = m.Content.Text
				} else if m.Content.HTML != "" {
					nm.Content = m.Content.HTML
				}

				isActive := false
				if m.Author.Name != "" {
					nm.Author = &services.EntryAuthor{}
					isActive = true
					nm.Author.Name = m.Author.Name
				}

				if m.Author.Photo != "" {
					if !isActive {
						nm.Author = &services.EntryAuthor{}
					}

					nm.Author.Photo = m.Author.Photo
				}

				if m.Author.URL != "" {
					if !isActive {
						nm.Author = &services.EntryAuthor{}
					}

					nm.Author.URL = m.Author.URL
				}

				newMentions = append(newMentions, nm)
			}

			id := strings.TrimSuffix(entry.ID, "/")
			id = strings.TrimPrefix(id, "/")
			id = strings.ReplaceAll(id, "/", "-")

			if id == "" {
				id = "index"
			}

			fmt.Println(id)

			dst := filepath.Join(c.Hugo.Source, "data", "interactions", id+".yaml")
			fmt.Println(dst)

			ttt, err := yaml.Marshal(newMentions)
			if err != nil {
				panic(err)
			}

			err = ioutil.WriteFile(dst, ttt, 0644)
			if err != nil {
				panic(err)
			}

			err = os.Remove(wmpath)
			if err != nil {
				panic(err)
			}
		}
	}
}

func testSave() {
	c, err := config.Parse()
	if err != nil {
		panic(err)
	}

	eagle, err := services.NewEagle(c)
	if err != nil {
		panic(err)
	}

	entries, err := eagle.GetAll()
	if err != nil {
		panic(err)
	}

	fmt.Printf("got %d entries\n", len(entries))

	for _, entry := range entries {
		entry.Path = ""
		err = eagle.SaveEntry(entry)
		if err != nil {
			panic(err)
		}
	}
}
