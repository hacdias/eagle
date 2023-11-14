package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"go.hacdias.com/eagle/config"
	"go.hacdias.com/eagle/log"
	"go.uber.org/zap"
)

const (
	AssetsBaseURL string = "/assets"
)

type Asset struct {
	Type      string
	Path      string
	Integrity string
	Body      []byte
}

func (r *Renderer) AssetByPath(path string) *Asset {
	return r.assets.byPath[path]
}

func (r *Renderer) AssetByName(name string) *Asset {
	return r.assets.byName[name]
}

type assetsBuilder struct {
	log    *zap.SugaredLogger
	fs     afero.Fs
	assets []config.Asset
	byName map[string]*Asset
	byPath map[string]*Asset
}

func newAssetsBuilder(source string, assets []config.Asset) *assetsBuilder {
	dir := filepath.Join(source, "assets")
	fs := afero.NewBasePathFs(afero.NewOsFs(), dir)

	return &assetsBuilder{
		log:    log.S().Named("assets"),
		fs:     fs,
		assets: assets,
		byName: map[string]*Asset{},
		byPath: map[string]*Asset{},
	}
}

func (b *assetsBuilder) build() error {
	paths := map[string]*Asset{}
	builds := map[string]*Asset{}

	for _, asset := range b.assets {
		parsedAsset, err := b.buildOne(&asset)
		if err != nil {
			return err
		}

		paths[asset.Name] = parsedAsset
		builds[parsedAsset.Path] = parsedAsset
		b.log.Debugw("asset built", "path", parsedAsset.Path, "integrity", parsedAsset.Integrity, "type", parsedAsset.Type)
	}

	b.byName = paths
	b.byPath = builds
	return nil
}

func (b *assetsBuilder) buildOne(asset *config.Asset) (*Asset, error) {
	var data bytes.Buffer

	// Combine all files into a single one
	for _, filename := range asset.Files {
		fd, err := b.fs.Open(filename)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		_, err = io.Copy(&data, fd)
		if err != nil {
			return nil, err
		}
	}

	var (
		ext         = filepath.Ext(asset.Name)
		name        = strings.TrimSuffix(asset.Name, ext)
		contentType string
	)

	// Determine content type
	switch ext {
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	}

	// Calculate hash
	raw := data.Bytes()
	sha256 := sha256.New()
	if _, err := sha256.Write(raw); err != nil {
		return nil, err
	}
	sha := sha256.Sum(nil)

	return &Asset{
		Type:      contentType,
		Path:      filepath.Join(AssetsBaseURL, fmt.Sprintf("%s.%x%s", name, sha, ext)),
		Integrity: "sha256-" + base64.StdEncoding.EncodeToString(sha),
		Body:      raw,
	}, nil
}
