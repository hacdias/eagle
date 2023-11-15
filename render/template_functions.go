package render

import (
	"html/template"
	"net/url"
	"time"
)

func (r *Renderer) getTemplateFuncMap(alwaysAbsolute bool) template.FuncMap {
	funcs := template.FuncMap{
		// "truncate":       util.TruncateStringWithEllipsis,
		// "domain":         util.Domain,
		// "humanDomain":    humanDomain,
		// "strContains":    strings.Contains,
		// "strSplit":       strings.Split,
		// "strJoin":        strings.Join,
		// "safeHTML":       safeHTML,
		// "safeCSS":        safeCSS,
		// "imageURL":       r.ResolveImageURL,
		// "imageSourceSet": r.ResolveImageSourceSet,
		// "dateFormat":     dateFormat,
		"now": time.Now,
		// "include":        r.getIncludeTemplate(alwaysAbsolute),
		// "md":             r.getRenderMarkdown(alwaysAbsolute),
		"absURL": absoluteURL(r.cfg.Site.BaseURL),
		"relURL": relativeURL(r.cfg.Site.BaseURL),
		// "stars":          stars,
		// "sprintf":        fmt.Sprintf,
		// "slugify":        util.Slugify,
	}

	if alwaysAbsolute {
		// funcs["relURL"] = r.c.Server.AbsoluteURL
	}

	return funcs
}

func resolvedURL(baseStr, refStr string) *url.URL {
	base, _ := url.Parse(baseStr)
	page, _ := url.Parse(refStr)
	return base.ResolveReference(page)
}

func absoluteURL(baseStr string) func(string) string {
	return func(refStr string) string {
		resolved := resolvedURL(baseStr, refStr)
		if resolved == nil {
			return ""
		}
		return resolved.String()
	}
}

func relativeURL(baseStr string) func(string) string {
	return func(refStr string) string {
		resolved := resolvedURL(baseStr, refStr)
		if resolved == nil {
			return refStr
		}

		// Take out everything before the path.
		resolved.User = nil
		resolved.Host = ""
		resolved.Scheme = ""
		return resolved.String()
	}
}
