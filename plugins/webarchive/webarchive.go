package webarchive

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

var (
	_ server.HookPlugin = &WebArchive{}
)

func init() {
	server.RegisterPlugin("webarchive", NewWebArchive)
}

type WebArchive struct {
	core   *core.Core
	fields []string
}

func NewWebArchive(co *core.Core, config map[string]any) (server.Plugin, error) {
	cfg := typed.New(config)

	var fields []string
	if cfg.String("fields") == "@all" {
		fields = []string{"bookmark-of"}
	} else if cfgFields, ok := cfg.StringsIf("fields"); ok {
		fields = cfgFields
	} else {
		return nil, errors.New("fields missing")
	}

	return &WebArchive{
		core:   co,
		fields: fields,
	}, nil
}

func (wa *WebArchive) PreSaveHook(*core.Entry) error {
	return nil
}

func (wa *WebArchive) PostSaveHook(e *core.Entry) error {
	var errs error

	other := typed.New(e.Other)
	for _, field := range wa.fields {
		url := other.String(field)
		if url == "" {
			continue
		}

		if archived := other.String("wa-" + field); archived != "" {
			continue
		}

		location, err := webArchive(url)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		e.Other["wa-"+field] = location
	}

	if err := wa.core.SaveEntry(e); err != nil {
		errs = errors.Join(errs, err)
	}

	return errs
}

func webArchive(url string) (string, error) {
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head("https://web.archive.org/save/" + url)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status code not ok: %d", resp.StatusCode)
	}

	location, err := resp.Location()
	if err != nil {
		return "", err
	}

	return location.String(), nil
}
