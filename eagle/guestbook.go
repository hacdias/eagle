package eagle

import "time"

type GuestbookEntry struct {
	Name    string    `json:"name,omitempty"`
	Website string    `json:"website,omitempty"`
	Content string    `json:"content,omitempty"`
	Date    time.Time `json:"date,omitempty"`
	Unseen  bool      `json:"unseen,omitempty"`
}

type GuestbookEntries []GuestbookEntry
