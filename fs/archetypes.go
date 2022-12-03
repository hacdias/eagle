package fs

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var defaultTemplate = template.Must(template.New("default").Parse(`---
published: {{ .Now.Format "2006-01-02T15:04:05Z07:00" }}
---

`))

func (f *FS) LoadArchetypes() (map[string]*template.Template, error) {
	archetypes := map[string]*template.Template{}

	files, err := f.ReadDir(ArchetypesDirectory)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	for _, file := range files {
		at, err := f.ReadFile(filepath.Join(ArchetypesDirectory, file.Name()))
		if err != nil {
			return nil, err
		}

		filename := file.Name()
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)

		tmpl, err := template.New(name).Parse(string(at))
		if err != nil {
			panic(err)
		}

		archetypes[name] = tmpl
	}

	if _, ok := archetypes["default"]; !ok {
		archetypes["default"] = defaultTemplate
	}

	return archetypes, nil
}
