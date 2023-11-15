package render

import (
	"html/template"

	"go.hacdias.com/eagle/config"
	"go.hacdias.com/eagle/entry"
)

type Alternate struct {
	Type string
	Href string
}

type Pagina struct {
	// set by caller of [Renderer.Render]
	*entry.Entry
	IsHome     bool
	IsList     bool
	Alternates []Alternate

	// set by [Renderer.Render]
	Site    config.SiteConfig
	Assets  Assets
	Content template.HTML

	// Me     eagle.User
	// // For page-specific variables.
	// Data interface{}
	// IsLoggedIn bool
	// NoIndex    bool
	// fs *fs.FS
}
