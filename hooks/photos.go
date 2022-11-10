package hooks

import (
	"reflect"
	"strings"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/media"
)

type PhotosProcessor struct {
	fs    *fs.FS
	media *media.Media
}

func NewPhotosProcessor(fs *fs.FS, media *media.Media) *PhotosProcessor {
	return &PhotosProcessor{
		fs:    fs,
		media: media,
	}
}

func (p *PhotosProcessor) EntryHook(e *eagle.Entry, isNew bool) error {
	return p.ProcessPhotos(e)
}

func (p *PhotosProcessor) ProcessPhotos(e *eagle.Entry) error {
	if e.Properties == nil {
		return nil
	}

	v, ok := e.Properties["photo"]
	if !ok {
		return nil
	}

	upload := func(url string) string {
		if strings.HasPrefix(url, "http") && !strings.HasPrefix(url, p.media.BaseURL()) {
			return p.media.SafeUploadFromURL("media", url, false)
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

	_, err := p.fs.TransformEntry(e.ID, func(ee *eagle.Entry) (*eagle.Entry, error) {
		ee.Properties["photo"] = newPhotos
		return ee, nil
	})

	return err
}
