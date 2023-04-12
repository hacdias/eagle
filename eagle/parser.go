package eagle

import (
	"errors"
	urlpkg "net/url"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type Parser struct {
	baseURL string
}

func NewParser(baseURL string) *Parser {
	return &Parser{baseURL: baseURL}
}

func (p *Parser) Parse(id, raw string) (*Entry, error) {
	id = cleanID(id)
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	permalink, err := p.makePermalink(id)
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
		FrontMatter: FrontMatter{},
	}

	fr := &FrontMatter{}
	err = yaml.Unmarshal([]byte(splits[0]), &fr)
	if err != nil {
		return nil, err
	}

	e.FrontMatter = *fr
	return e, nil
}

func (p *Parser) makePermalink(id string) (string, error) {
	url, err := urlpkg.Parse(p.baseURL)
	if err != nil {
		return "", err
	}
	url.Path = id
	return url.String(), nil
}
