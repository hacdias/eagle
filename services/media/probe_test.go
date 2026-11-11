package media

import (
	"os"
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

func TestProbeRoundsFractions(t *testing.T) {
	// The fixture, with FNumber 1244236/699009 and FocalLength 251773/37217,
	// as written by Photos.
	data, err := os.ReadFile("testdata/odd-rationals.jpeg")
	if err != nil {
		t.Fatal(err)
	}

	_, photo, err := probe(data)
	if err != nil {
		t.Fatal(err)
	}

	if photo.FocalLength != 6.76 || photo.Aperture != 1.78 {
		t.Errorf("got focal length %v and aperture %v, want 6.76 and 1.78", photo.FocalLength, photo.Aperture)
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
