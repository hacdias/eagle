package render

import (
	"html/template"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

const (
	// TemplatesDirectory string = "templates"

	LayoutError  string = "error"
	LayoutIndex  string = "index"
	LayoutList   string = "list"
	LayoutSingle string = "single"
	LayoutTerms  string = "terms"
	// TemplateFeed      string = "feed"
	// TemplateLogin     string = "login"
	// TemplateSearch    string = "search"
	// TemplateNew       string = "new"
	// TemplateEditor    string = "editor"
	// TemplateAuth      string = "auth"
	// TemplateDashboard string = "dashboard"
)

type templatesBuilder struct {
	dir string
	fs  *afero.Afero
}

func newTemplatesBuilder(source string) *templatesBuilder {
	return &templatesBuilder{
		fs:  &afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(source, "templates"))},
		dir: filepath.Join(source, "templates"),
	}
}

func (b *templatesBuilder) loadPartials(fns template.FuncMap) (*template.Template, error) {
	partials := template.New("").Funcs(fns)

	err := b.fs.Walk("partials", func(filepath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		name := strings.TrimPrefix(filepath, "partials"+afero.FilePathSeparator)
		name = strings.TrimSuffix(name, ".html")
		name = path.Clean(name)

		fileContent, err := b.fs.ReadFile(filepath)
		if err != nil {
			return err
		}

		partials, err = partials.New(name).Parse(string(fileContent))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return partials, nil
}

func (b *templatesBuilder) loadLayouts(fns template.FuncMap) (map[string]*template.Template, error) {
	partials, err := b.loadPartials(fns)
	if err != nil {
		return nil, err
	}

	baseTemplate, err := b.fs.ReadFile("layouts/baseof.html")
	if err != nil {
		return nil, err
	}

	layouts := map[string]*template.Template{}

	err = b.fs.Walk("layouts", func(filepath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		name := strings.TrimPrefix(filepath, "layouts"+afero.FilePathSeparator)
		name = strings.TrimSuffix(name, ".html")
		name = path.Clean(name)

		tpl, err := partials.Clone()
		if err != nil {
			return err
		}

		tpl, err = tpl.New(name).Parse(string(baseTemplate))
		if err != nil {
			return err
		}

		fileContent, err := b.fs.ReadFile(filepath)
		if err != nil {
			return err
		}

		tpl, err = tpl.Parse(string(fileContent))
		if err != nil {
			return err
		}

		layouts[name] = tpl
		return nil
	})

	return layouts, err
}

func (b *templatesBuilder) load(fns template.FuncMap) (map[string]*template.Template, error) {
	// TODO: shortcodes
	return b.loadLayouts(fns)
}
