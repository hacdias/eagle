package eagle

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/contenttype"
)

const (
	AssetsBaseURL string = "/assets"
)

type Asset struct {
	Type string
	Body []byte
}

func (e *Eagle) BuildAssets() error {
	assets, err := e.getAssets()
	if err != nil {
		return err
	}
	e.assets = assets
	return nil
}

func (e *Eagle) getAssets() (map[string]string, error) {
	assets := map[string]string{}
	for _, asset := range e.Config.Assets {
		data, err := e.getAsset(asset)
		if err != nil {
			return nil, err
		}

		filename, err := e.saveAsset(filepath.Ext(asset.Name), data)
		if err != nil {
			return nil, err
		}

		assets[asset.Name] = filename
	}

	return assets, nil
}

func (e *Eagle) saveAsset(ext string, data []byte) (string, error) {
	sha256 := sha256.New()
	if _, err := sha256.Write(data); err != nil {
		return "", err
	}

	// This is where the asset will be available at.
	path := filepath.Join(AssetsBaseURL, fmt.Sprintf("%x%s", sha256.Sum(nil), ext))
	return path, e.SaveCache(path, data)
}

func (e *Eagle) getAsset(asset *config.Asset) ([]byte, error) {
	var data bytes.Buffer

	for _, file := range asset.Files {
		filename := filepath.Join(AssetsDirectory, file)
		raw, err := e.fs.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		_, err = data.Write(raw)
		if err != nil {
			return nil, err
		}
	}

	var contentType string

	switch filepath.Ext(asset.Name) {
	case ".css":
		contentType = contenttype.CSS
	case ".js":
		contentType = contenttype.JS
	default:
		return data.Bytes(), nil
	}

	var w bytes.Buffer

	err := e.minifier.Minify(contentType, &w, &data)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}
