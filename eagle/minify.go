package eagle

import (
	"github.com/hacdias/eagle/v4/contenttype"
	"github.com/tdewolff/minify/v2"
	mCss "github.com/tdewolff/minify/v2/css"
	mHtml "github.com/tdewolff/minify/v2/html"
	mJs "github.com/tdewolff/minify/v2/js"
	mJson "github.com/tdewolff/minify/v2/json"
	mXml "github.com/tdewolff/minify/v2/xml"
)

func initMinifier() *minify.M {
	m := minify.New()
	m.AddFunc(contenttype.HTML, mHtml.Minify)
	m.AddFunc(contenttype.CSS, mCss.Minify)
	m.AddFunc(contenttype.XML, mXml.Minify)
	m.AddFunc(contenttype.JS, mJs.Minify)
	m.AddFunc(contenttype.RSS, mXml.Minify)
	m.AddFunc(contenttype.ATOM, mXml.Minify)
	m.AddFunc(contenttype.JSONFeed, mJson.Minify)
	m.AddFunc(contenttype.AS, mJson.Minify)
	return m
}
