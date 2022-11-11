package hooks

import (
	"reflect"
	"strings"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/media"
)

type PhotosUploader struct {
	fs    *fs.FS
	media *media.Media
}

func NewPhotosUploader(fs *fs.FS, media *media.Media) *PhotosUploader {
	return &PhotosUploader{
		fs:    fs,
		media: media,
	}
}

func (p *PhotosUploader) EntryHook(e *eagle.Entry, isNew bool) error {
	if e.Listing != nil {
		return nil
	}

	return p.ProcessPhotos(e)
}

func (p *PhotosUploader) ProcessPhotos(e *eagle.Entry) error {
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