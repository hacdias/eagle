package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/yaml"
	"github.com/karlseguin/typed"
	"go.uber.org/zap"
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
		Domain:        c.Domain,
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
		// moveKey(entry.Metadata, "date", "publishDate")
		// moveKey(entry.Metadata, "lastmod", "updateDate")
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

type HugoEntry struct {
	ID        string
	Permalink string
	// RawContent is the content that comes directly from the
	// Micropub request. It is populated with .Content on all
	// other situations.
	RawContent string
	Content    string
	Metadata   typed.Typed
	Listing    bool
}

type Hugo struct {
	*zap.SugaredLogger
	config.Hugo
	Domain        string
	DirChanges    chan string
	currentSubDir string
}

func generateHash() string {
	h := fnv.New64a()
	// the implementation does not return errors
	_, _ = h.Write([]byte(time.Now().UTC().String()))
	return hex.EncodeToString(h.Sum(nil))
}

// ShouldBuild should only be called on startup to make sure there's
// a built public directory to serve.
func (h *Hugo) ShouldBuild() (bool, error) {
	if h.currentSubDir != "" {
		return false, nil
	}

	content, err := ioutil.ReadFile(path.Join(h.Destination, "last"))
	if err != nil {
		if !os.IsNotExist(err) {
			return true, nil
		}

		return true, err
	}

	h.currentSubDir = string(content)
	h.DirChanges <- filepath.Join(h.Destination, h.currentSubDir)
	return false, nil
}

func (h *Hugo) Build(clean bool) error {
	if h.currentSubDir == "" {
		_, err := h.ShouldBuild()
		if err != nil {
			return err
		}
	}

	dir := h.currentSubDir
	new := dir == "" || clean

	if new {
		dir = generateHash()
	}

	destination := filepath.Join(h.Destination, dir)
	args := []string{"--minify", "--destination", destination}

	cmd := exec.Command("hugo", args...)
	cmd.Dir = h.Source
	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("hugo run failed: %s: %s", err, out)
	}

	if new {
		// We build to a different sub directory so we can change the directory
		// we are serving seamlessly without users noticing. Check server/satic.go!
		h.currentSubDir = dir
		h.DirChanges <- filepath.Join(h.Destination, h.currentSubDir)
		err = ioutil.WriteFile(path.Join(h.Destination, "last"), []byte(dir), 0644)
		if err != nil {
			return fmt.Errorf("could not write last dir: %s", err)
		}
	}

	return nil
}

func (h *Hugo) makeURL(id string) (string, error) {
	u, err := url.Parse(h.Domain)
	if err != nil {
		return "", err
	}
	u.Path = id
	return u.String(), nil
}

func (h *Hugo) cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}

func (h *Hugo) GetEntry(id string) (*HugoEntry, error) {
	id = h.cleanID(id)
	index := filepath.Join(h.Source, "content", id)
	list := false

	if _, err := os.Stat(filepath.Join(index, "_index.md")); os.IsNotExist(err) {
		index = filepath.Join(index, "index.md")
	} else if err != nil {
		return nil, err
	} else {
		list = true
		index = filepath.Join(index, "_index.md")
	}

	bytes, err := ioutil.ReadFile(index)
	if err != nil {
		return nil, err
	}

	splits := strings.SplitN(string(bytes), "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	permalink, err := h.makeURL(id)
	if err != nil {
		return nil, err
	}

	entry := &HugoEntry{
		ID:        id,
		Permalink: permalink,
		Metadata:  map[string]interface{}{},
		Content:   strings.TrimSpace(splits[1]),
		Listing:   list,
	}

	entry.RawContent = entry.Content

	var metadata map[string]interface{}

	err = yaml.Unmarshal([]byte(splits[0]), &metadata)
	if err != nil {
		return nil, err
	}

	entry.Metadata = metadata

	if props, ok := entry.Metadata["properties"]; ok {
		entry.Metadata["properties"] = internalToMf2(props)
	} else {
		entry.Metadata["properties"] = map[string][]interface{}{}
	}

	return entry, nil
}

func (h *Hugo) GetAll() ([]*HugoEntry, error) {
	entries := []*HugoEntry{}
	content := path.Join(h.Source, "content")

	err := filepath.Walk(h.Source, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() != "index.md" && info.Name() != "_index.md" {
			return nil
		}

		id := strings.TrimPrefix(path.Dir(p), content)
		entry, err := h.GetEntry(id)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}

func mf2ToInternal(data interface{}) interface{} {
	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		if value.Len() == 1 {
			return mf2ToInternal(value.Index(0).Interface())
		}

		parsed := make([]interface{}, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = mf2ToInternal(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string]interface{}{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			parsed[fmt.Sprint(k.Interface())] = mf2ToInternal(v.Interface())
		}

		return parsed
	}

	return data
}

func internalToMf2(data interface{}) interface{} {
	if data == nil {
		return []interface{}{nil}
	}

	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		parsed := make([]interface{}, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = internalToMf2(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string][]interface{}{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			key := fmt.Sprint(k.Interface())
			vk := reflect.TypeOf(v.Interface()).Kind()

			if key == "properties" || key == "value" || vk == reflect.Slice || vk == reflect.Array {
				parsed[key] = internalToMf2(v.Interface()).([]interface{})
			} else {
				parsed[key] = []interface{}{internalToMf2(v.Interface())}
			}
		}

		return parsed
	}

	return data
}
