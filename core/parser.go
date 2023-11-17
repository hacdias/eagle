package core

import (
	"errors"
	"fmt"
	urlpkg "net/url"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// TODO: do not hardcore this. Instead, use Hugo's configuration to deduce
// and "back-engineer" how the permalinks are constructed. Then this can be used
// only in the parser code.
const SpecialSection = "posts"

type Parser struct {
	baseURL string
}

func NewParser(baseURL string) *Parser {
	return &Parser{baseURL: baseURL}
}

func (p *Parser) Parse(id, raw string) (*Entry, error) {
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	fr := &FrontMatter{}
	err := yaml.Unmarshal([]byte(splits[0]), &fr)
	if err != nil {
		return nil, err
	}

	id = cleanID(id)

	permalink, err := p.makePermalink(id, fr)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(splits[1])
	if content != "" {
		// Fixes issue where goldmark is adding a <blockquote>
		// if the document ends with an HTML tag.
		content += "\n"
	}

	e := &Entry{
		ID:          id,
		Permalink:   permalink,
		Content:     content,
		FrontMatter: *fr,
	}

	return e, nil
}

func (p *Parser) makePermalink(id string, fr *FrontMatter) (string, error) {
	url, err := urlpkg.Parse(p.baseURL)
	if err != nil {
		return "", err
	}

	if fr.URL != "" {
		url.Path = cleanID(fr.URL)
		return url.String(), nil
	}

	parts := strings.Split(id, "/")
	if parts[1] == SpecialSection && !fr.Date.IsZero() {
		url.Path = fmt.Sprintf("/%04d/%02d/%02d/%s/", fr.Date.Year(), fr.Date.Month(), fr.Date.Day(), parts[len(parts)-2])
	} else {
		url.Path = id
	}

	return url.String(), nil
}
