package static

import (
	"embed"
)

//go:embed "js/*"
//go:embed "css/*"
//go:embed "favicon.ico"
var FS embed.FS
