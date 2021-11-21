package database

import (
	"time"

	"github.com/hacdias/eagle/v2/entry"
)

type Read struct {
	ID     string    `json:"id"`
	Date   time.Time `json:"date"`
	Name   string    `json:"name"`
	Author string    `json:"author"`
}

type ReadsSummary struct {
	ToRead   []*Read `json:"to-read"`
	Reading  []*Read `json:"reading"`
	Finished []*Read `json:"finished"`
}

type Watch struct {
	ID   string    `json:"id"`
	Date time.Time `json:"date"`
	Name string    `json:"name"`
}

type WatchesSummary struct {
	Series []*Watch `json:"series"`
	Movies []*Watch `json:"movies"`
}

type PaginationOptions struct {
	Page  int
	Limit int
}

type QueryOptions struct {
	PaginationOptions
	Draft   bool
	Deleted bool
	Private bool
}

type Database interface {
	Close()

	Remove(id string)
	Add(...*entry.Entry) error

	GetTags() ([]string, error)
	Search(opt *QueryOptions, query string) ([]string, error)
	ByTag(opt *QueryOptions, tag string) ([]string, error)
	BySection(opt *QueryOptions, sections ...string) ([]string, error)
	ByDate(opts *QueryOptions, year, month, day int) ([]string, error)

	GetDeleted(opts *PaginationOptions) ([]string, error)
	GetDrafts(opts *PaginationOptions) ([]string, error)

	ReadsSummary() (*ReadsSummary, error)
	WatchesSummary() (*WatchesSummary, error)
	Been() ([]string, error)
	SectionsCount() (map[string]int, error)
}
