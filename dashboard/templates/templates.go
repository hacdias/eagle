package templates

import "embed"

//go:embed *.html
var FS embed.FS

//go:embed base.html
var Base string
