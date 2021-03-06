package static

import "embed"

//go:embed "js/*"
//go:embed "css/*"
var FS embed.FS
