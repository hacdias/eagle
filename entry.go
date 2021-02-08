package eagle

import "time"

// [pictures emoji description syndication replyTo mentions aliases expiryDate tags math noIndex url layout date lastmod mermaid title noMentions]
type Entry struct {
	Path string // might be empty

	ID        string
	Permalink string
	Content   string
	Metadata  EntryMetadata
}

type EntryMetadata struct {
	Title       string        `yaml:"title,omitempty"`
	Description string        `yaml:"description,omitempty"`
	Tags        []string      `yaml:"tags,omitempty"`
	Date        time.Time     `yaml:"date,omitempty"`
	Lastmod     time.Time     `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time     `yaml:"expiryDate,omitempty"`
	Syndication []string      `yaml:"syndication,omitempty"`
	ReplyTo     EmbeddedEntry `yaml:"replyTo,omitempty"`
	URL         string        `yaml:"url,omitempty"`
	Aliases     []string      `yaml:"aliases,omitempty"`
	Emoji       string        `yaml:"emoji,omitempty"`
	Layout      string        `yaml:"layout,omitempty"`
	NoIndex     bool          `yaml:"noIndex,omitempty"`
	NoMentions  bool          `yaml:"noMentions,omitempty"`
	Math        bool          `yaml:"math,omitempty"`
	Mermaid     bool          `yaml:"mermaid,omitempty"`
}

type EmbeddedEntry struct {
	WmID    uint      `yaml:"wm-id,omitempty"`
	Type    string    `yaml:"type,omitempty"`
	URL     string    `yaml:"url,omitempty"`
	Name    string    `yaml:"name,omitempty"`
	Content string    `yaml:"content,omitempty"`
	Date    time.Time `yaml:"date,omitempty"`
	Author  *Author   `yaml:"author,omitempty"`
}

type Author struct {
	Name  string `yaml:"name,omitempty"`
	URL   string `yaml:"url,omitempty"`
	Photo string `yaml:"photo,omitempty"`
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
