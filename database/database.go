package database

import (
	"github.com/hacdias/eagle/v3/entry"
)

type PaginationOptions struct {
	Page  int
	Limit int
}

type QueryOptions struct {
	PaginationOptions
	WithDrafts  bool
	WithDeleted bool

	// Empty matches all visibilities.
	Visibility []entry.Visibility

	// Empty matches all audiences.
	Audience string
}

type Database interface {
	Close()

	Remove(id string)
	Add(...*entry.Entry) error

	GetTags() ([]string, error)
	GetEmojis() ([]string, error)
	Search(opt *QueryOptions, query string) ([]string, error)

	GetDeleted(opts *PaginationOptions) ([]string, error)
	GetDrafts(opts *PaginationOptions) ([]string, error)
	GetUnlisted(opts *PaginationOptions) ([]string, error)
	GetPrivate(opts *PaginationOptions, audience string) ([]string, error)

	GetAll(opts *QueryOptions) ([]string, error)

	ByTag(opt *QueryOptions, tag string) ([]string, error)
	ByEmoji(opt *QueryOptions, emoji string) ([]string, error)
	BySection(opt *QueryOptions, sections ...string) ([]string, error)
	ByDate(opts *QueryOptions, year, month, day int) ([]string, error)
	ByProperty(opts *QueryOptions, property, value string) ([]string, error)

	ReadsSummary() (*entry.ReadsSummary, error)
	WatchesSummary() (*entry.WatchesSummary, error)
	Been() ([]string, error)
	SectionsCount() (map[string]int, error)
}
