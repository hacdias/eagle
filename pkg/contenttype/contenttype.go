package contenttype

// Inspired from:
// https://git.jlel.se/jlelse/GoBlog

const (
	CharsetUtf8Suffix = "; charset=utf-8"

	AS            = "application/activity+json"
	ATOM          = "application/atom+xml"
	CSS           = "text/css"
	HTML          = "text/html"
	JS            = "application/javascript"
	JSON          = "application/json"
	JSONFeed      = "application/feed+json"
	LDJSON        = "application/ld+json"
	MultipartForm = "multipart/form-data"
	RSS           = "application/rss+xml"
	Text          = "text/plain"
	WWWForm       = "application/x-www-form-urlencoded"
	XML           = "text/xml"

	ASUTF8   = AS + CharsetUtf8Suffix
	CSSUTF8  = CSS + CharsetUtf8Suffix
	HTMLUTF8 = HTML + CharsetUtf8Suffix
	JSONUTF8 = JSON + CharsetUtf8Suffix
	JSUTF8   = JS + CharsetUtf8Suffix
	TextUTF8 = Text + CharsetUtf8Suffix
	XMLUTF8  = XML + CharsetUtf8Suffix
)
