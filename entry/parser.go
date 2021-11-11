package entry

import (
	"errors"
	urlpkg "net/url"
	"strings"
)

type Parser struct {
	baseURL string
}

func NewParser(baseURL string) *Parser {
	return &Parser{baseURL: baseURL}
}

func (p *Parser) FromMF2(mf2Data map[string][]interface{}, slug string) (*Entry, error) {
	entry := &Entry{
		Frontmatter: Frontmatter{},
	}

	err := entry.Update(mf2Data)
	if err != nil {
		return nil, err
	}

	id := newID(slug, entry.Published)
	entry.ID = cleanID(id)
	entry.Permalink, err = p.makePermalink(entry.ID)
	return entry, err
}

func (p *Parser) FromRaw(id, raw string) (*Entry, error) {
	id = cleanID(id)
	splits := strings.SplitN(raw, "\n---", 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	permalink, err := p.makePermalink(id)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		ID:          id,
		Permalink:   permalink,
		Content:     strings.TrimSpace(splits[1]),
		Frontmatter: Frontmatter{},
	}

	fr, err := unmarshalFrontmatter([]byte(splits[0]))
	if err != nil {
		return nil, err
	}

	entry.Frontmatter = *fr
	return entry, nil
}

func (p *Parser) makePermalink(id string) (string, error) {
	url, err := urlpkg.Parse(p.baseURL)
	if err != nil {
		return "", err
	}
	url.Path = id
	return url.String(), nil
}
