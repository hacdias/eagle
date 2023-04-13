package indexer

import (
	"io"
	"os"

	"github.com/hacdias/eagle/core"
)

type Pagination struct {
	Page  int
	Limit int
}

type Query struct {
	Pagination

	WithDrafts   bool
	WithDeleted  bool
	WithUnlisted bool
}

type Backend interface {
	io.Closer

	Add(...*core.Entry) error
	Remove(ids ...string)

	GetDrafts(opts *Pagination) ([]string, error)
	GetUnlisted(opts *Pagination) ([]string, error)
	GetDeleted(opts *Pagination) ([]string, error)
	GetSearch(opt *Query, query string) ([]string, error)
	GetCount() (int, error)

	ClearEntries()
}

type Indexer struct {
	backend Backend
	fs      *core.FS
}

func NewIndexer(fs *core.FS, backend Backend) *Indexer {
	return &Indexer{
		fs:      fs,
		backend: backend,
	}
}

func (e *Indexer) Add(entries ...*core.Entry) error {
	return e.backend.Add(entries...)
}

func (e *Indexer) Remove(ids ...string) {
	e.backend.Remove(ids...)
}

func (e *Indexer) GetDrafts(opts *Pagination) (core.Entries, error) {
	return e.idsToEntries(e.backend.GetDrafts(opts))
}

func (e *Indexer) GetUnlisted(opts *Pagination) (core.Entries, error) {
	return e.idsToEntries(e.backend.GetUnlisted(opts))
}

func (e *Indexer) GetDeleted(opts *Pagination) (core.Entries, error) {
	return e.idsToEntries(e.backend.GetDeleted(opts))
}

func (e *Indexer) GetSearch(opts *Query, query string) (core.Entries, error) {
	return e.idsToEntries(e.backend.GetSearch(opts, query))
}

func (e *Indexer) GetCount() (int, error) {
	return e.backend.GetCount()
}

func (e *Indexer) ClearEntries() {
	e.backend.ClearEntries()
}

func (e *Indexer) Close() error {
	return e.backend.Close()
}

func (e *Indexer) idsToEntries(ids []string, err error) (core.Entries, error) {
	if err != nil {
		return nil, err
	}

	entries := core.Entries{}

	for _, id := range ids {
		entry, err := e.fs.GetEntry(id)
		if err != nil {
			if os.IsNotExist(err) {
				e.backend.Remove(id)
			} else {
				return nil, err
			}
		} else {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
