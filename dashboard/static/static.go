package static

import (
	"embed"
)

//go:embed "js/*"
//go:embed "css/*"
//go:embed "favicon.png"
var FS embed.FS
