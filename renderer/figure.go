package renderer

import (
	"bytes"
	"io"
	urlpkg "net/url"
	"strings"

	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

func (r *Renderer) GetPictureURL(urlStr, size, format string) string {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return ""
	}

	query := url.Query()
	query.Del("class")
	query.Del("id")
	query.Del("caption")
	url.RawQuery = query.Encode()

	if url.Scheme == "cdn" && r.mediaBaseURL != "" {
		id := strings.TrimPrefix(url.Path, "/")
		return r.getCdnURL(id, size, format)
	} else {
		return url.String()
	}
}

func (r *Renderer) getCdnSourceSet(id, format string) string {
	return r.getCdnURL(id, "250", format) + " 250w" +
		", " + r.getCdnURL(id, "500", format) + " 500w" +
		", " + r.getCdnURL(id, "1000", format) + " 1000w" +
		", " + r.getCdnURL(id, "2000", format) + " 2000w"
}

func (r *Renderer) getCdnURL(id, size, format string) string {
	return r.mediaBaseURL + "/img/" + size + "/" + id + "." + format
}

type figureWriter interface {
	io.Writer
	WriteByte(c byte) error
	WriteString(s string) (int, error)
	WriteRune(r rune) (size int, err error)
}

// Syntax
//
//	![Alt text](url "Title")
//	url?class=my+class									--> Add class.
//	url?id=someid												--> Add id.
//	url?caption=false							  		--> Do not print "Title" as <figcaption>.
//
// URL should be either:
//   - cdn:/slug-at-cdn									--> Renders <figure> with many <source>.
//   - /relative/to/image.jpeg					--> Renders an <img> by default.
//   - http://example.com/example.jpg		-->	Renders an <img> by default.
func (r *Renderer) writeFigure(w figureWriter, imgURL, alt, title string, absURLs, unsafe, uPhoto bool) error {
	url, err := urlpkg.Parse(imgURL)
	if err != nil {
		return err
	}

	query := url.Query()

	_, _ = w.WriteString("<figure")

	if class := query.Get("class"); class != "" {
		query.Del("class")
		_, _ = w.WriteString(" class=\"")
		_, _ = w.WriteString(class)
		_ = w.WriteByte('"')
	}

	if id := query.Get("id"); id != "" {
		query.Del("id")
		_, _ = w.WriteString(" id=\"")
		_, _ = w.WriteString(id)
		_ = w.WriteByte('"')
	}

	caption := true
	if c := query.Get("caption"); c != "" {
		caption = c == "true"
		query.Del("caption")
	}

	_ = w.WriteByte('>')

	url.RawQuery = query.Encode()

	var imgSrc []byte

	_, _ = w.WriteString("<picture>")

	if url.Scheme == "cdn" && r.mediaBaseURL != "" {
		id := strings.TrimPrefix(url.Path, "/")
		imgSrc = []byte(r.getCdnURL(id, "2000", "jpeg"))

		_, _ = w.WriteString("<source srcset=\"")
		_, _ = w.WriteString(r.getCdnSourceSet(id, "webp"))
		_, _ = w.WriteString("\" type=\"image/webp\">")

		_, _ = w.WriteString("<source srcset=\"")
		_, _ = w.WriteString(r.getCdnSourceSet(id, "jpeg"))
		_, _ = w.WriteString("\">")
	} else {
		imgSrc = []byte(url.String())
	}

	_, _ = w.WriteString("<img src=\"")
	if absURLs && r.c.Server.BaseURL != "" && bytes.HasPrefix(imgSrc, []byte("/")) {
		_, _ = w.Write(util.EscapeHTML([]byte(r.c.Server.BaseURL)))
	}
	if unsafe || !html.IsDangerousURL(imgSrc) {
		_, _ = w.Write(util.EscapeHTML(imgSrc))
	}
	_, _ = w.WriteRune('"')

	if uPhoto {
		_, _ = w.WriteString(" class=\"u-photo\"")
	}

	if alt != "" {
		_, _ = w.WriteString(` alt="`)
		_, _ = w.Write(util.EscapeHTML([]byte(alt)))
		_, _ = w.WriteRune('"')
	}
	_, _ = w.WriteString(" loading=\"lazy\">")
	_, _ = w.WriteString("</picture>")

	if caption && title != "" {
		_, _ = w.WriteString("<figcaption>")
		_, _ = w.Write(util.EscapeHTML([]byte(title)))
		_, _ = w.WriteString("</figcaption>")
	}

	_, _ = w.WriteString("</figure>")
	return nil
}
