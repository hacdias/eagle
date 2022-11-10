package eagle

import "time"

type Watch struct {
	ID   string    `json:"id"`
	Date time.Time `json:"date"`
	Name string    `json:"name"`
}

type WatchesSummary struct {
	Series []*Watch `json:"series"`
	Movies []*Watch `json:"movies"`
}
