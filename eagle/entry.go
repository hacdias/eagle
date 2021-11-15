package eagle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/entry/mf2"
	"github.com/thoas/go-funk"
)

type EntryTransformer func(*entry.Entry) (*entry.Entry, error)

func (e *Eagle) GetEntry(id string) (*entry.Entry, error) {
	filepath, err := e.guessPath(id)
	if err != nil {
		return nil, err
	}

	raw, err := e.fs.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	entry, err := e.Parser.FromRaw(id, string(raw))
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (e *Eagle) GetEntries() ([]*entry.Entry, error) {
	entries := []*entry.Entry{}
	err := e.fs.Walk(ContentDirectory, func(p string, info os.FileInfo, err error) error {
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

func (e *Eagle) SaveEntry(entry *entry.Entry) error {
	e.entriesMu.Lock()
	defer e.entriesMu.Unlock()

	return e.saveEntry(entry)
}

func (e *Eagle) PostSaveEntry(ee *entry.Entry, syndicators []string) {
	// Check for context URL and fetch the data if needed.
	err := e.ensureContextXRay(ee)
	if err != nil {
		e.Error(err)
	}

	// Uploads photos if they exist. This may change the entry.
	err = e.processPhotos(ee)
	if err != nil {
		e.Error(err)
	}

	// TODO: Check if the post has a 'location' Geo URI and parse it.

	// TODO: If it is a checkin, download location map.

	// Syndicate. This may change the entry.
	err = e.syndicate(ee, syndicators)
	if err != nil {
		e.Error(err)
	}

	// Remove entry from the cache. Every other action from here on
	// should not influence how the entry is rendered.
	e.RemoveCache(ee)

	// Send webmentions.
	err = e.SendWebmentions(ee)
	if err != nil {
		e.Error(err)
	}
}

func (e *Eagle) processPhotos(ee *entry.Entry) error {
	if ee.Properties == nil {
		return nil
	}

	v, ok := ee.Properties["photo"]
	if !ok {
		return nil
	}

	upload := func(url string) string {
		if strings.HasPrefix(url, "http") && !strings.HasPrefix(url, e.Config.BunnyCDN.Base) {
			return e.safeUploadFromURL("/u", url)
		}
		return url
	}

	var newPhotos interface{}

	if vv, ok := v.(string); ok {
		newPhotos = upload(vv)
	} else {
		value := reflect.ValueOf(v)
		kind := value.Kind()
		parsed := []interface{}{}

		if kind != reflect.Array && kind != reflect.Slice {
			return nil
		}

		for i := 0; i < value.Len(); i++ {
			v = value.Index(i).Interface()

			if vv, ok := v.(string); ok {
				parsed = append(parsed, upload(vv))
			} else if vv, ok := v.(map[string]interface{}); ok {
				if value, ok := vv["value"].(string); ok {
					vv["value"] = upload(value)
				}
				parsed = append(parsed, vv)
			}
		}

		newPhotos = parsed
	}

	_, err := e.TransformEntry(ee.ID, func(ee *entry.Entry) (*entry.Entry, error) {
		ee.Properties["photo"] = newPhotos
		return ee, nil
	})

	return err
}

func (e *Eagle) TransformEntry(id string, transformers ...EntryTransformer) (*entry.Entry, error) {
	if len(transformers) == 0 {
		return nil, errors.New("at least one entry transformer must be provided")
	}

	// TODO: instead, I could keep the file open for writing
	// and avoid the need for locks entirely.
	// However, take into account that .SaveEntry uses
	// .entryCheck to fill missing fields.
	e.entriesMu.Lock()
	defer e.entriesMu.Unlock()

	ee, err := e.GetEntry(id)
	if err != nil {
		return nil, err
	}

	for _, t := range transformers {
		ee, err = t(ee)
		if err != nil {
			return nil, err
		}
	}

	err = e.saveEntry(ee)
	return ee, err
}

func EntryTemplates(ee *entry.Entry) []string {
	tpls := []string{}
	if ee.Template != "" {
		tpls = append(tpls, ee.Template)
	}
	tpls = append(tpls, TemplateSingle)
	return tpls
}

func (e *Eagle) saveEntry(entry *entry.Entry) error {
	err := e.entryCheck(entry)
	if err != nil {
		return err
	}

	path, err := e.guessPath(entry.ID)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// Default path for new files is content/{slug}/index.md
		path = filepath.Join(ContentDirectory, entry.ID, "index.md")
	}

	err = e.fs.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}

	str, err := entry.String()
	if err != nil {
		return err
	}

	err = e.fs.WriteFile(path, []byte(str), "update "+entry.ID)
	if err != nil {
		return fmt.Errorf("could not save entry: %w", err)
	}

	_ = e.db.Add(entry)
	return nil
}

func (e *Eagle) entryCheck(entry *entry.Entry) error {
	mm := entry.Helper()
	postType := mm.PostType()

	if funk.Contains(e.allowedTypes, postType) {
		// Only add the sections to entries under the /year/month/date.
		// This avoids adding sections to top-level pages that shouldn't
		// have these sections.
		if len(entry.Sections) == 0 && strings.HasPrefix(entry.ID, "/20") {
			entry.Sections = e.Config.Site.MicropubTypes[postType]
		}
	} else {
		return errors.New("type not supported " + string(postType))
	}

	if entry.Description == "" {
		switch mm.PostType() {
		case mf2.TypeCheckin:
			checkin := mm.Sub(mm.TypeProperty())
			if name := checkin.Name(); name != "" {
				entry.Description = "At " + name
			}
		}
	}

	// TODO: perhaps I only want better pictures in the /photos section. Since I want to show
	// pictures on the homepage, this would get noisy. The other option is to change the rules
	// for home, like "accept photos that are not checkins".
	//
	// if len(entry.Helper().Photos()) > 0 && !funk.ContainsString(entry.Sections, "photos") {
	// 	entry.Sections = append(entry.Sections, "photos")
	// }

	return nil
}

func (e *Eagle) guessPath(id string) (string, error) {
	path := filepath.Join(ContentDirectory, id, "index.md")
	_, err := e.fs.Stat(path)
	if err == nil {
		return path, nil
	}

	return "", err
}

func (e *Eagle) ensureContextXRay(ee *entry.Entry) error {
	mm := ee.Helper()
	typ := mm.PostType()

	if typ != mf2.TypeLike && typ != mf2.TypeRepost && typ != mf2.TypeReply {
		return nil
	}

	urlStr := mm.String(mm.TypeProperty())
	if urlStr == "" {
		return fmt.Errorf("expected context url to be non-empty for %s", ee.ID)
	}

	sidecar, err := e.GetSidecar(ee)
	if err != nil {
		return fmt.Errorf("could not fetch sidecar for %s: %w", ee.ID, err)
	}

	if sidecar.Context != nil {
		return nil
	}

	context, err := e.getXRay(urlStr)
	if err != nil {
		return fmt.Errorf("could not fetch context xray for %s: %w", ee.ID, err)
	}

	return e.UpdateSidecar(ee, func(data *Sidecar) (*Sidecar, error) {
		data.Context = context
		return data, nil
	})
}

func (e *Eagle) syndicate(ee *entry.Entry, syndicators []string) error {
	if len(syndicators) == 0 {
		return nil
	}

	syndications, err := e.syndication.Syndicate(ee, syndicators)
	if err != nil {
		return err
	}

	_, err = e.TransformEntry(ee.ID, func(ee *entry.Entry) (*entry.Entry, error) {
		mm := ee.Helper()
		syndications := append(mm.Strings("syndication"), syndications...)
		ee.Properties["syndication"] = syndications
		return ee, nil
	})
	return err
}
