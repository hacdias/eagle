package server

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/services/media"
)

func TestUpdateEntryWithPhotos(t *testing.T) {
	if _, err := exec.LookPath("vips"); err != nil {
		t.Skip("vips is not installed")
	}

	s := newTestServer(t)
	require.NoError(t, s.initMediaCache())

	conf := &core.Media{}
	conf.Storage.FileSystem = &core.FileSystem{Base: "http://localhost", Directory: t.TempDir()}
	s.media = media.NewMedia(conf)

	// The camera details are only in the uploaded file, which is stripped of them.
	data, err := os.ReadFile("../services/media/testdata/exif.jpeg")
	require.NoError(t, err)
	s.mediaCache.Set("cache:x.jpeg", data)

	e := &core.Entry{ID: "/2026/11/10/test/"}
	e.Date = time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	e.Photos = []core.Photo{{URL: "cache:x.jpeg", Title: "Title", Alt: "Alt"}}

	require.NoError(t, s.updateEntryWithPhotos(e))
	assert.Equal(t, core.Photo{
		URL:          "http://localhost/2026-11-10-test.jpeg",
		Title:        "Title",
		Alt:          "Alt",
		Width:        48,
		Height:       64,
		Camera:       "Canon EOS R5",
		Lens:         "RF24-105mm F4 L IS USM",
		FocalLength:  50,
		Aperture:     4,
		ShutterSpeed: "1/250",
		ISO:          200,
	}, e.Photos[0])
}
