package services

import "time"

type Entry struct {
	Path      string // The original path of the file. Might be empty.
	ID        string
	Permalink string
	Content   string
	Metadata  EntryMetadata
}

type EntryMetadata struct {
	Title       string         `yaml:"title,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	Date        time.Time      `yaml:"date,omitempty"`
	Lastmod     time.Time      `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time      `yaml:"expiryDate,omitempty"`
	Syndication []string       `yaml:"syndication,omitempty"`
	ReplyTo     *EmbeddedEntry `yaml:"replyTo,omitempty"`
	URL         string         `yaml:"url,omitempty"`
	Aliases     []string       `yaml:"aliases,omitempty"`
	Emoji       string         `yaml:"emoji,omitempty"`
	Layout      string         `yaml:"layout,omitempty"`
	NoIndex     bool           `yaml:"noIndex,omitempty"`
	NoMentions  bool           `yaml:"noMentions,omitempty"`
	Math        bool           `yaml:"math,omitempty"`
	Mermaid     bool           `yaml:"mermaid,omitempty"`
	Pictures    []EntryPicture `yaml:"pictures,omitempty"`
	Mentions    []EntryMention `yaml:"mentions,omitempty"`
}

type EmbeddedEntry struct {
	WmID    int          `yaml:"wm-id,omitempty"`
	Type    string       `yaml:"type,omitempty"`
	URL     string       `yaml:"url,omitempty"`
	Name    string       `yaml:"name,omitempty"`
	Content string       `yaml:"content,omitempty"`
	Date    time.Time    `yaml:"date,omitempty"`
	Author  *EntryAuthor `yaml:"author,omitempty"`
}

type EntryPicture struct {
	Title string `yaml:"title,omitempty"`
	Slug  string `yaml:"slug,omitempty"`
	Hide  bool   `yaml:"hide,omitempty"`
}

type EntryMention struct {
	Href string `yaml:"href,omitempty"`
	Name string `yaml:"name,omitempty"`
}

type EntryAuthor struct {
	Name  string `yaml:"name,omitempty" json:"name"`
	URL   string `yaml:"url,omitempty" json:"url"`
	Photo string `yaml:"photo,omitempty" json:"photo"`
}

/*
file := id + ".md"
	isList := false
	raw, err := e.afs.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			file = id + "/_index.md"
			raw, err = e.afs.ReadFile(file)
			isList = true
		}

		if err != nil {
			return nil, err
		}
	}

	splits := bytes.SplitN(raw, []byte("\n---"), 2)
	if len(splits) != 2 {
		return nil, errors.New("could not parse file: splits !== 2")
	}

	content := bytes.TrimSpace(splits[1])

	entry := &eagle.Entry{
		ID:         id,
		Metadata:   eagle.EntryMetadata{},
		RawContent: content,
		Content:    content, // TODO
		Permalink:  "http://hacdias.com" + id,
		Section:    strings.Split(strings.TrimLeft(id, "/"), "/")[0], // TODO: check
		IsList:     isList,
	}

	err = yaml.Unmarshal(splits[0], &entry.Metadata)
	if err != nil {
		return nil, err
	}

	return entry, nil
*/
