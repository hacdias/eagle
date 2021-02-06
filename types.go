package eagle

import "time"

type Entry struct {
	ID       string
	Content  []byte
	Metadata EntryMetadata
}

type EntryMetadata struct {
	Title       string    `yaml:"title,omitempty"`
	Description string    `yaml:"description,omitempty"`
	Tags        []string  `yaml:"tags,omitempty"`
	PublishDate time.Time `yaml:"publishDate,omitempty"`
	UpdateDate  time.Time `yaml:"updateDate,omitempty"`
	ExpiryDate  time.Time `yaml:"expiryDate,omitempty"`
	Syndication []string  `yaml:"syndication,omitempty"`
	Emoji       string    `yaml:"emoji,omitempty"`
	Layout      string    `yaml:"layout,omitempty"`

	NoIndex    bool `yaml:"noIndex,omitempty"`
	NoMentions bool `yaml:"noMentions,omitempty"`

	Math    bool `yaml:"math,omitempty"`
	Mermaid bool `yaml:"mermaid,omitempty"`
}
