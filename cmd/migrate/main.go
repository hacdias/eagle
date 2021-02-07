package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/yaml"
	"github.com/karlseguin/typed"
)

const DST = "./_DATA/content" // trailing!

func main() {
	c, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		_ = c.L().Sync()
	}()

	migrate(c)
}

func migrate(c *config.Config) {
	hugo := &Hugo{
		SugaredLogger: c.S().Named("hugo"),
		Hugo:          c.Hugo,
		Domain:        c.Site.Domain,
	}

	os.RemoveAll(DST)
	os.MkdirAll(DST, 0777)

	entries, err := hugo.GetAll()
	if err != nil {
		log.Fatal(err)
	}

	keys := map[string]bool{}
	aliases := ""

	for _, entry := range entries {
		//src := path.Join(c.Hugo.Source, "content", entry.ID, "index.md")
		dst := DST + entry.ID + ".md"
		if entry.Listing {
			dst = DST + entry.ID + "/_index.md"
		}

		if props, ok := entry.Metadata["properties"].(map[string][]interface{}); ok {
			if len(reflect.ValueOf(props).MapKeys()) > 0 {
				// Reply
				reply := props["in-reply-to"]
				if len(reply) > 0 {
					if len(reply) > 1 {
						log.Panic(fmt.Errorf("post repllies to more than one thing %s", reply))
					}

					rp := reply[0].(string)
					mfile := fmt.Sprintf("%x.json", sha256.Sum256([]byte(rp)))
					mfilep := path.Join(c.Hugo.Source, "data", "xray", mfile)

					if _, err := os.Stat(mfilep); err == nil || os.IsExist(err) {
						entry.Metadata["replyTo"] = rp
					} else {
						log.Fatal(err)
					}
				}
				delete(props, "in-reply-to")

				syndication := props["syndication"]
				if len(syndication) > 0 {
					entry.Metadata["syndication"] = syndication
				}
				delete(props, "syndication")

				if len(reflect.ValueOf(props).MapKeys()) > 0 {
					log.Fatal(fmt.Errorf("prop not recognized: %v", reflect.ValueOf(props).MapKeys()))
				}
			}

			delete(entry.Metadata, "properties")
		}

		delete(entry.Metadata, "home")
		delete(entry.Metadata, "type")
		delete(entry.Metadata, "menu")
		moveKey(entry.Metadata, "date", "publishDate")
		moveKey(entry.Metadata, "lastmod", "updateDate")
		moveKey(entry.Metadata, "hideMentions", "noMentions")
		moveKey(entry.Metadata, "noindex", "noIndex")

		if alias, ok := entry.Metadata.StringsIf("aliases"); ok {
			for _, a := range alias {
				aliases += fmt.Sprintf("%s %s\n", a, entry.ID)
			}
			delete(entry.Metadata, "aliases")
		}

		for key := range entry.Metadata {
			keys[key] = true
		}

		err = os.MkdirAll(path.Dir(dst), 0777)
		if err != nil {
			log.Fatal(err)
		}

		err = saveEntry(entry, dst)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = ioutil.WriteFile(path.Join(DST, "../redirects"), []byte(aliases), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed: %d\n", len(entries))
	fmt.Printf("Keys: %v\n", reflect.ValueOf(keys).MapKeys())
}

func saveEntry(e *HugoEntry, dst string) error {
	val, err := yaml.Marshal(&e.Metadata)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dst, []byte(fmt.Sprintf("---\n%s---\n\n%s", string(val), e.Content)), 0644)
	if err != nil {
		return fmt.Errorf("could not save entry: %s", err)
	}

	return nil
}

func moveKey(m typed.Typed, from, to string) {
	if v, ok := m[from]; ok {
		m[to] = v
		delete(m, from)
	}
}
