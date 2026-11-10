package media

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/bep/imagemeta"
	"github.com/gabriel-vasile/mimetype"
	"go.hacdias.com/eagle/core"
)

var imageFormats = map[string]imagemeta.ImageFormat{
	"image/jpeg": imagemeta.JPEG,
	"image/png":  imagemeta.PNG,
	"image/webp": imagemeta.WebP,
}

// probe returns the format of an image, such as "jpeg", and a photo with its
// upright size and camera details.
//
// shortcut: imagemeta skips WebP EXIF that starts with "Exif\x00\x00", as vips
// writes it, so such WebP lose camera details and rotation; upgrade once
// imagemeta accepts that prefix.
func probe(data []byte) (string, *core.Photo, error) {
	mime := mimetype.Detect(data).String()
	imageFormat, ok := imageFormats[mime]
	if !ok {
		return "", nil, fmt.Errorf("unsupported image type %q", mime)
	}

	var (
		photo        core.Photo
		maker, model string
		orientation  uint16
	)

	result, err := imagemeta.Decode(imagemeta.Options{
		R:           bytes.NewReader(data),
		ImageFormat: imageFormat,
		Sources:     imagemeta.CONFIG | imagemeta.EXIF,
		HandleTag: func(tag imagemeta.TagInfo) error {
			rat, _ := tag.Value.(imagemeta.Rat[uint32])

			switch tag.Tag {
			case "Make":
				maker, _ = tag.Value.(string)
			case "Model":
				model, _ = tag.Value.(string)
			case "LensModel":
				photo.Lens, _ = tag.Value.(string)
			case "Orientation":
				orientation, _ = tag.Value.(uint16)
			case "ISO":
				iso, _ := tag.Value.(uint16)
				photo.ISO = int(iso)
			case "FocalLength":
				if rat != nil {
					photo.FocalLength = rat.Float64()
				}
			case "FNumber":
				if rat != nil {
					photo.Aperture = rat.Float64()
				}
			case "ExposureTime":
				if rat != nil {
					photo.ShutterSpeed = rat.String()
				}
			}

			return nil
		},
	})
	if err != nil {
		return "", nil, err
	}

	photo.Width, photo.Height = result.ImageConfig.Width, result.ImageConfig.Height

	// Orientations 5 to 8 turn the image sideways.
	if orientation >= 5 {
		photo.Width, photo.Height = photo.Height, photo.Width
	}

	photo.Camera = cameraName(maker, model)
	return strings.TrimPrefix(mime, "image/"), &photo, nil
}

// cameraName joins the make and model, unless the model already names the
// make, such as "Canon EOS R5" or "NIKON Z 6".
func cameraName(maker, model string) string {
	brand, _, _ := strings.Cut(maker, " ")
	if brand == "" || strings.Contains(strings.ToLower(model), strings.ToLower(brand)) {
		return model
	}
	return strings.TrimSpace(maker + " " + model)
}
