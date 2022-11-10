package renderer

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/pkg/contenttype"
)

const (
	AssetsBaseURL string = "/assets"
)

type Assets struct {
	paths  map[string]string
	assets map[string]*Asset
}

func (a *Assets) Path(id string) string {
	return a.paths[id]
}

func (a *Assets) Get(id string) *Asset {
	return a.assets[id]
}

type Asset struct {
	Type string
	Path string
	Body []byte
}

func (r *Renderer) LoadAssets() error {
	assets, err := r.getAssets()
	if err != nil {
		return err
	}
	r.assets = assets
	return nil
}

func (r *Renderer) GetAssets() *Assets {
	return r.assets
}

func (r *Renderer) getAssets() (*Assets, error) {
	assets := &Assets{
		paths:  map[string]string{},
		assets: map[string]*Asset{},
	}

	for _, asset := range r.c.Source.Assets {
		parsedAsset, err := r.buildAsset(&asset)
		if err != nil {
			return nil, err
		}

		assets.paths[asset.Name] = parsedAsset.Path
		assets.assets[parsedAsset.Path] = parsedAsset
	}

	return assets, nil
}

func (r *Renderer) buildAsset(asset *eagle.Asset) (*Asset, error) {
	var data bytes.Buffer

	for _, file := range asset.Files {
		filename := filepath.Join(fs.AssetsDirectory, file)
		raw, err := r.fs.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		_, err = data.Write(raw)
		if err != nil {
			return nil, err
		}
	}

	var (
		ext         = filepath.Ext(asset.Name)
		contentType string
		raw         []byte
	)

	switch ext {
	case ".css":
		contentType = contenttype.CSS
	case ".js":
		contentType = contenttype.JS
	default:
		raw = data.Bytes()
	}

	if contentType != "" {
		var w bytes.Buffer

		err := r.minify.Minify(contentType, &w, &data)
		if err != nil {
			return nil, err
		}

		raw = w.Bytes()
	}

	sha256 := sha256.New()
	if _, err := sha256.Write(raw); err != nil {
		return nil, err
	}

	// This is where the asset will be available at.
	path := filepath.Join(AssetsBaseURL, fmt.Sprintf("%x%s", sha256.Sum(nil), ext))

	return &Asset{
		Type: contentType,
		Path: path,
		Body: raw,
	}, nil
}
