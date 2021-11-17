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

type ReadsStatistics struct {
	ToRead   []*Read `json:"to-read"`
	Reading  []*Read `json:"reading"`
	Finished []*Read `json:"finished"`
}

type QueryOptions struct {
	Page    int
	Limit   int
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

	ReadsStatistics() (*ReadsStatistics, error)
}
