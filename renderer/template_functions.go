package renderer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	urlpkg "net/url"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/util"
	"github.com/samber/lo"
	"github.com/yuin/goldmark/renderer/html"
	gutil "github.com/yuin/goldmark/util"
)

func (r *Renderer) getIncludeTemplate(absoluteURLs bool) func(name string, data ...interface{}) (template.HTML, error) {
	return func(name string, data ...interface{}) (template.HTML, error) {
		var templates map[string]*template.Template
		if absoluteURLs {
			templates = r.absoluteTemplates
		} else {
			templates = r.templates
		}

		var (
			buf bytes.Buffer
			err error
		)

		if len(data) == 1 {
			err = templates[name].ExecuteTemplate(&buf, name, data[0])
		} else if len(data) == 2 {
			// TODO: perhaps make more type verifications.
			nrd := *data[0].(*RenderData)
			listing := nrd.Entry.Listing
			nrd.Entry = data[1].(*eagle.Entry)
			nrd.Entry.Listing = listing
			nrd.sidecar = nil
			err = templates[name].ExecuteTemplate(&buf, name, &nrd)
		} else {
			return "", errors.New("wrong parameters")
		}

		return template.HTML(buf.String()), err
	}
}

func humanDomain(text string) string {
	u, err := urlpkg.Parse(text)
	if err != nil {
		return text
	}

	if strings.Contains(u.Host, "reddit.com") {
		parts := strings.Split(u.Path, "/")
		if len(parts) >= 3 {
			return "r/" + parts[2]
		}
	}

	return u.Host
}

func safeHTML(text string) template.HTML {
	return template.HTML(text)
}

func safeCSS(text string) template.CSS {
	return template.CSS(text)
}

func asJSON(a interface{}) string {
	data, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(data)
}

func dateFormat(date, template string) string {
	t, err := dateparse.ParseStrict(date)
	if err != nil {
		return date
	}
	return t.Format(template)
}

func stars(rating, total int) template.HTML {
	stars := ""

	for i := 0; i < total; i++ {
		if i < rating {
			stars += "★"
		} else {
			stars += "☆"
		}
	}

	return template.HTML(stars)
}

func durationFromSeconds(seconds float64) time.Duration {
	return time.Second * time.Duration(seconds)
}

func (r *Renderer) getRenderMarkdown(absoluteURLs bool) func(string) template.HTML {
	if absoluteURLs {
		return r.RenderAbsoluteMarkdown
	} else {
		return r.RenderRelativeMarkdown
	}
}

func (r *Renderer) getUPhoto(absoluteURLs bool) func(url, alt string) template.HTML {
	return func(urlStr, alt string) template.HTML {
		var w strings.Builder

		url, err := urlpkg.Parse(urlStr)
		if err != nil {
			return template.HTML("")
		}

		_, _ = w.WriteString("<figure>")
		_, _ = w.WriteString("<picture>")

		var imgSrc []byte
		if url.Scheme == "cdn" && r.m != nil {
			id := strings.TrimPrefix(url.Path, "/")
			imgSrc = []byte(r.m.ImageURL(id))

			for format, srcset := range r.m.ImageSourceSet(id) {
				_, _ = w.WriteString("<source srcset=\"")
				_, _ = w.WriteString(srcset)
				_, _ = w.WriteString("\" type=\"image/")
				_, _ = w.WriteString(format)
				_, _ = w.WriteString("\">")
			}
		} else {
			imgSrc = []byte(url.String())
		}

		_, _ = w.WriteString("<img src=\"")
		if absoluteURLs && r.c.Server.BaseURL != "" && bytes.HasPrefix(imgSrc, []byte("/")) {
			_, _ = w.Write(gutil.EscapeHTML([]byte(r.c.Server.BaseURL)))
		}
		if !html.IsDangerousURL(imgSrc) {
			_, _ = w.Write(gutil.EscapeHTML(imgSrc))
		}
		_, _ = w.WriteRune('"')
		_, _ = w.WriteString(" class=\"u-photo\"")

		if alt != "" {
			_, _ = w.WriteString(` alt="`)
			_, _ = w.Write(gutil.EscapeHTML([]byte(alt)))
			_, _ = w.WriteRune('"')
		}

		_, _ = w.WriteString(" loading=\"lazy\">")
		_, _ = w.WriteString("</picture>")
		_, _ = w.WriteString("</figure>")
		return template.HTML(w.String())
	}
}

func (r *Renderer) getTemplateFuncMap(alwaysAbsolute bool) template.FuncMap {
	funcs := template.FuncMap{
		"truncate":            util.TruncateStringWithEllipsis,
		"domain":              util.Domain,
		"humanDomain":         humanDomain,
		"strContains":         strings.Contains,
		"strSplit":            strings.Split,
		"strJoin":             strings.Join,
		"containsString":      lo.Contains[string],
		"safeHTML":            safeHTML,
		"safeCSS":             safeCSS,
		"uPhoto":              r.getUPhoto(alwaysAbsolute),
		"figureURL":           r.ImageURL,
		"dateFormat":          dateFormat,
		"now":                 time.Now,
		"include":             r.getIncludeTemplate(alwaysAbsolute),
		"md":                  r.getRenderMarkdown(alwaysAbsolute),
		"absURL":              r.c.Server.AbsoluteURL,
		"relURL":              r.c.Server.RelativeURL,
		"stars":               stars,
		"sprintf":             fmt.Sprintf,
		"durationFromSeconds": durationFromSeconds,
		"asJSON":              asJSON,
		"slugify":             util.Slugify,
	}

	if alwaysAbsolute {
		funcs["relURL"] = r.c.Server.AbsoluteURL
	}

	return funcs
}
