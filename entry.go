package eagle

import "time"

type Entry struct {
	ID        string
	Permalink string
	Content   []byte
	Metadata  EntryMetadata
}

type EntryMetadata struct {
	Title       string    `yaml:"title,omitempty"`
	Description string    `yaml:"description,omitempty"`
	Tags        []string  `yaml:"tags,omitempty"`
	Date        time.Time `yaml:"date,omitempty"`
	Lastmod     time.Time `yaml:"lastmod,omitempty"`
	ExpiryDate  time.Time `yaml:"expiryDate,omitempty"`
	Syndication []string  `yaml:"syndication,omitempty"`
	ReplyTo     string    `yaml:"replyTo,omitempty"`

	Emoji  string `yaml:"emoji,omitempty"`
	Layout string `yaml:"layout,omitempty"`

	NoIndex    bool `yaml:"noIndex,omitempty"`
	NoMentions bool `yaml:"noMentions,omitempty"`

	Math    bool `yaml:"math,omitempty"`
	Mermaid bool `yaml:"mermaid,omitempty"`
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
