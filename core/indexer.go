package core

import (
	"io"
	"os"
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

type IndexerBackend interface {
	io.Closer

	Add(...*Entry) error
	Remove(ids ...string)

	GetDrafts(opts *Pagination) ([]string, error)
	GetUnlisted(opts *Pagination) ([]string, error)
	GetDeleted(opts *Pagination) ([]string, error)
	GetSearch(opt *Query, query string) ([]string, error)
	GetCount() (int, error)

	ClearEntries()
}

type Indexer struct {
	backend IndexerBackend
	fs      *FS
}

func NewIndexer(fs *FS, backend IndexerBackend) *Indexer {
	return &Indexer{
		fs:      fs,
		backend: backend,
	}
}

func (e *Indexer) Add(entries ...*Entry) error {
	return e.backend.Add(entries...)
}

func (e *Indexer) Remove(ids ...string) {
	e.backend.Remove(ids...)
}

func (e *Indexer) GetDrafts(opts *Pagination) (Entries, error) {
	return e.idsToEntries(e.backend.GetDrafts(opts))
}

func (e *Indexer) GetUnlisted(opts *Pagination) (Entries, error) {
	return e.idsToEntries(e.backend.GetUnlisted(opts))
}

func (e *Indexer) GetDeleted(opts *Pagination) (Entries, error) {
	return e.idsToEntries(e.backend.GetDeleted(opts))
}

func (e *Indexer) GetSearch(opts *Query, query string) (Entries, error) {
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

func (e *Indexer) idsToEntries(ids []string, err error) (Entries, error) {
	if err != nil {
		return nil, err
	}

	entries := Entries{}

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
