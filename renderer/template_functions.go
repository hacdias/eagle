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
	"github.com/thoas/go-funk"
)

func (r *Renderer) includeTemplate(name string, data ...interface{}) (template.HTML, error) {
	var (
		buf bytes.Buffer
		err error
	)

	if len(data) == 1 {
		err = r.templates[name].ExecuteTemplate(&buf, name, data[0])
	} else if len(data) == 2 {
		// TODO: perhaps make more type verifications.
		nrd := *data[0].(*RenderData)
		listing := nrd.Entry.Listing
		nrd.Entry = data[1].(*eagle.Entry)
		nrd.Entry.Listing = listing
		nrd.sidecar = nil
		err = r.templates[name].ExecuteTemplate(&buf, name, &nrd)
	} else {
		return "", errors.New("wrong parameters")
	}

	return template.HTML(buf.String()), err
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
		return func(source string) template.HTML {
			var buffer bytes.Buffer
			_ = r.absoluteMarkdown.Convert([]byte(source), &buffer)
			return template.HTML(buffer.Bytes())
		}
	} else {
		return func(source string) template.HTML {
			var buffer bytes.Buffer
			_ = r.markdown.Convert([]byte(source), &buffer)
			return template.HTML(buffer.Bytes())
		}
	}
}

func (r *Renderer) getTemplateFuncMap(alwaysAbsolute bool) template.FuncMap {
	figure := func(url, alt string, uPhoto bool) template.HTML {
		var w strings.Builder
		err := r.writeFigure(&w, url, alt, "", alwaysAbsolute, true, uPhoto)
		if err != nil {
			return template.HTML("")
		}
		return template.HTML(w.String())
	}

	funcs := template.FuncMap{
		"truncate":            util.TruncateStringWithEllipsis,
		"domain":              util.Domain,
		"humanDomain":         humanDomain,
		"strContains":         strings.Contains,
		"strSplit":            strings.Split,
		"strJoin":             strings.Join,
		"containsString":      funk.ContainsString,
		"safeHTML":            safeHTML,
		"safeCSS":             safeCSS,
		"figure":              figure,
		"figureURL":           r.getPictureURL,
		"dateFormat":          dateFormat,
		"now":                 time.Now,
		"include":             r.includeTemplate,
		"md":                  r.getRenderMarkdown(alwaysAbsolute),
		"absURL":              r.c.Server.AbsoluteURL,
		"relURL":              r.c.Server.RelativeURL,
		"stars":               stars,
		"sprintf":             fmt.Sprintf,
		"durationFromSeconds": durationFromSeconds,
		"asJSON":              asJSON,
		"slugify":             util.Slugify,
		"groupByFirstChar":    util.GroupByFirstChar,
	}

	if alwaysAbsolute {
		funcs["relURL"] = r.c.Server.AbsoluteURL
	}

	return funcs
}