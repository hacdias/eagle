package media

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"image/png"
	"math/rand/v2"
	"os"
	"os/exec"
	"testing"

	"github.com/bep/imagemeta"
	"github.com/gabriel-vasile/mimetype"
)

func requireVips(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("vips"); err != nil {
		t.Skip("vips is not installed")
	}
}

// fixture returns a 64x48 JPEG, turned sideways by its orientation, with camera
// details, a GPS location and a serial number in its metadata.
func fixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/exif.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// noiseJPEG returns a JPEG of random pixels, which compresses poorly. With
// metadata, it carries the metadata segments of the fixture.
func noiseJPEG(t *testing.T, width, height int, metadata bool) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := range img.Pix {
		img.Pix[i] = uint8(rand.IntN(256))
	}

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100})
	if err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	if !metadata {
		return data
	}

	// The APPn segments right after the start marker hold the metadata.
	meta := fixture(t)
	end := 2
	for meta[end+1] >= 0xE0 && meta[end+1] <= 0xEF {
		end += 2 + int(binary.BigEndian.Uint16(meta[end+2:]))
	}

	return append(append(data[:2:2], meta[2:end]...), data[2:]...)
}

func assertNoMetadata(t *testing.T, name string, data []byte) {
	t.Helper()

	var tags []string
	_, err := imagemeta.Decode(imagemeta.Options{
		R:           bytes.NewReader(data),
		ImageFormat: imageFormats[mimetype.Detect(data).String()],
		Sources:     imagemeta.EXIF | imagemeta.XMP | imagemeta.IPTC,
		HandleTag: func(tag imagemeta.TagInfo) error {
			tags = append(tags, tag.Tag)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(tags) > 0 {
		t.Errorf("%s: metadata not removed: %v", name, tags)
	}
}

func TestTransform(t *testing.T) {
	requireVips(t)

	// The fixture is displayed as 48x64.
	data := fixture(t)

	for _, tc := range []struct {
		format string
		width  int
		want   int
	}{
		{"jpeg", 0, 48}, // original size, upright
		{"webp", 32, 32},
		{"jpeg", 100, 48}, // not enlarged
	} {
		out, err := transform(data, tc.format, tc.width, 80, 0)
		if err != nil {
			t.Fatal(err)
		}

		format, photo, err := probe(out)
		if err != nil {
			t.Fatal(err)
		}
		if format != tc.format || photo.Width != tc.want || photo.Height <= photo.Width {
			t.Errorf("%s %d: got %s %dx%d, want upright %s %d wide", tc.format, tc.width, format, photo.Width, photo.Height, tc.format, tc.want)
		}

		assertNoMetadata(t, tc.format, out)
	}
}

func TestTransformColourProfile(t *testing.T) {
	requireVips(t)

	p3, err := run(fixture(t), "vips", "icc_transform", "stdin", ".jpeg[keep=icc]", "p3", "--input-profile", "srgb")
	if err != nil {
		t.Fatal(err)
	}

	// Originals keep their colour profile, resized images are converted to sRGB.
	for width, want := range map[int]bool{0: true, 32: false} {
		out, err := transform(p3, "jpeg", width, 80, 0)
		if err != nil {
			t.Fatal(err)
		}

		if got := bytes.Contains(out, []byte("ICC_PROFILE")); got != want {
			t.Errorf("width %d: got colour profile %t, want %t", width, got, want)
		}
	}
}

func TestTransformMaxBytes(t *testing.T) {
	requireVips(t)

	data := noiseJPEG(t, 800, 800, false)

	full, err := transform(data, "jpeg", 800, 100, 0)
	if err != nil {
		t.Fatal(err)
	}

	maxBytes := len(full) / 2
	capped, err := transform(data, "jpeg", 800, 100, maxBytes)
	if err != nil {
		t.Fatal(err)
	}

	if len(capped) > maxBytes {
		t.Errorf("got %d bytes, want at most %d", len(capped), maxBytes)
	}
}

func TestTransformTransparentToJPEG(t *testing.T) {
	requireVips(t)

	var buf bytes.Buffer
	err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 64, 64)))
	if err != nil {
		t.Fatal(err)
	}

	out, err := transform(buf.Bytes(), "jpeg", 64, 80, 0)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}

	if r, g, b, _ := img.At(0, 0).RGBA(); r>>8 < 250 || g>>8 < 250 || b>>8 < 250 {
		t.Errorf("transparency filled with (%d, %d, %d), want white", r>>8, g>>8, b>>8)
	}
}
