package media

import (
	"testing"

	"go.hacdias.com/eagle/core"
)

func TestProbe(t *testing.T) {
	format, photo, err := probe(fixture(t))
	if err != nil {
		t.Fatal(err)
	}

	want := core.Photo{
		Width:        48,
		Height:       64,
		Camera:       "Canon EOS R5",
		Lens:         "RF24-105mm F4 L IS USM",
		FocalLength:  50,
		Aperture:     4,
		ShutterSpeed: "1/250",
		ISO:          200,
	}
	if format != "jpeg" || *photo != want {
		t.Errorf("got %q %+v, want \"jpeg\" %+v", format, *photo, want)
	}
}

func TestCameraName(t *testing.T) {
	for _, tc := range []struct{ maker, model, want string }{
		{"FUJIFILM", "X-T5", "FUJIFILM X-T5"},
		{"Apple", "iPhone 16 Pro", "Apple iPhone 16 Pro"},
		{"Canon", "Canon EOS R5", "Canon EOS R5"},
		{"NIKON CORPORATION", "NIKON Z 6", "NIKON Z 6"},
		{"", "", ""},
	} {
		if got := cameraName(tc.maker, tc.model); got != tc.want {
			t.Errorf("cameraName(%q, %q) = %q, want %q", tc.maker, tc.model, got, tc.want)
		}
	}
}
