package eagle

import (
	"path"
	"path/filepath"
	"strings"
)

func (e *Eagle) getEntryHTML(entry *Entry) ([]byte, error) {
	filename := entry.ID
	if !strings.HasSuffix(filename, ".html") {
		filename = path.Join(filename, "index.html")
	}

	e.buildMu.Lock()
	defer e.buildMu.Unlock()

	filename = filepath.Join(e.currentPublicDir, filename)
	return e.dstFs.ReadFile(filename)
}
