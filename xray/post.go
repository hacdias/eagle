package xray

import (
	"time"

	"go.hacdias.com/indielib/microformats"
)

type Post struct {
	Name        string            `json:"name,omitempty"`
	Content     string            `json:"content,omitempty"`
	Author      string            `json:"author,omitempty"`
	AuthorPhoto string            `json:"authorPhoto,omitempty"`
	AuthorURL   string            `json:"authorUrl,omitempty"`
	Date        time.Time         `json:"date,omitempty"`
	URL         string            `json:"url,omitempty"`
	Type        microformats.Type `json:"type,omitempty"`
	Private     bool              `json:"private,omitempty"`
}
