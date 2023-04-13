package xray

import (
	"time"
)

type Author struct {
	Name  string `json:"name,omitempty"`
	Photo string `json:"photo,omitempty"`
	URL   string `json:"url,omitempty"`
}

type Post struct {
	Author    Author    `json:"author,omitempty"`
	Published time.Time `json:"published,omitempty"`
	Name      string    `json:"name,omitempty"`
	Content   string    `json:"content,omitempty"`
	URL       string    `json:"url,omitempty"`
	Private   bool      `json:"private,omitempty"`
}
