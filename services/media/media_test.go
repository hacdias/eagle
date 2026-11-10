package media

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.hacdias.com/eagle/core"
)

func TestUploadImage(t *testing.T) {
	requireVips(t)

	dir := t.TempDir()
	conf := &core.Media{}
	conf.Storage.FileSystem = &core.FileSystem{Base: "http://localhost", Directory: dir}
	m := NewMedia(conf)

	// Small WebP, without the fixture's metadata, which imagemeta cannot read
	// when written by vips.
	webp, err := run(fixture(t), "vips", "copy", "stdin", ".webp[keep=none]")
	if err != nil {
		t.Fatal(err)
	}

	// Low quality JPEG, which would grow when encoded at a higher quality.
	lowQuality, err := transform(noiseJPEG(t, 800, 600, false), "jpeg", 0, 40, 0)
	if err != nil {
		t.Fatal(err)
	}

	var icon bytes.Buffer
	err = png.Encode(&icon, image.NewNRGBA(image.Rect(0, 0, 16, 16)))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name   string
		data   []byte
		url    string
		stored string
		width  int
		height int
		camera string
	}{
		// Large: 400x300 pixels, turned sideways by the fixture's orientation.
		{"photo.jpeg", noiseJPEG(t, 400, 300, true), "image:photo", "photo.jpeg", 300, 400, "Canon EOS R5"},
		{"low.jpeg", lowQuality, "image:low", "low.jpeg", 800, 600, ""},
		{"graphic.webp", webp, "http://localhost/graphic.webp", "graphic.webp", 64, 48, ""},
		{"icon.png", icon.Bytes(), "http://localhost/icon.png", "icon.png", 16, 16, ""},
	} {
		ext := filepath.Ext(tc.name)
		_, photo, err := m.UploadMedia(strings.TrimSuffix(tc.name, ext), ext, bytes.NewReader(tc.data))
		if err != nil {
			t.Fatal(err)
		}

		if photo.URL != tc.url || photo.Width != tc.width || photo.Height != tc.height || photo.Camera != tc.camera {
			t.Errorf("%s: got %+v", tc.name, *photo)
		}

		stored, err := os.ReadFile(filepath.Join(dir, tc.stored))
		if err != nil {
			t.Fatal(err)
		}

		assertNoMetadata(t, tc.name, stored)
		if ext != ".png" && len(stored) > len(tc.data) {
			t.Errorf("%s: stored %d bytes, more than the %d uploaded", tc.name, len(stored), len(tc.data))
		}
	}

	for _, format := range formats {
		for _, width := range widths {
			_, err := os.Stat(filepath.Join(dir, "image", strconv.Itoa(int(width)), "photo."+string(format)))
			if err != nil {
				t.Error(err)
			}
		}
	}
}
