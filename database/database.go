package database

import (
	"github.com/hacdias/eagle/v2/entry"
)

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
}
