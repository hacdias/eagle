package media

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// transform encodes an image in the given format with vips, removing all
// metadata. With a width, the image is resized to it and converted to sRGB.
// Without, it keeps its size and colour profile. With maxBytes, the quality is
// lowered until the image fits.
func transform(data []byte, format string, width, quality, maxBytes int) ([]byte, error) {
	options := "keep=none"
	if width == 0 {
		options = "keep=icc"
	}
	if format == "jpeg" {
		// JPEG has no transparency: fill it with white, as vips defaults to black.
		options += ",interlace,optimize_coding,background=255"
	}

	for {
		target := fmt.Sprintf(".%s[Q=%d,%s]", format, quality, options)

		// autorot reads from the "stdin" file name, thumbnail_source from a descriptor.
		args := []string{"autorot", "stdin", target}
		if width > 0 {
			// The height is unbounded so that only the width constrains the size.
			args = []string{"thumbnail_source", "[descriptor=0]", target, strconv.Itoa(width),
				"--height", "100000", "--size", "down", "--output-profile", "srgb"}
		}

		out, err := run(data, "vips", args...)
		if err != nil {
			return nil, err
		}

		if maxBytes == 0 || len(out) <= maxBytes || quality <= 10 {
			return out, nil
		}

		quality -= 10
	}
}

func run(stdin []byte, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = &stderr
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, stderr.String())
	}

	return out, nil
}
