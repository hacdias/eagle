package core

import (
	"time"
)

type Notifier interface {
	Info(msg string)
	Error(err error)
}

type GuestbookEntry struct {
	ID      string    `json:"-"`
	Name    string    `json:"name,omitempty"`
	Website string    `json:"website,omitempty"`
	Content string    `json:"content,omitempty"`
	Date    time.Time `json:"date,omitempty"`
}

type GuestbookEntries []GuestbookEntry
