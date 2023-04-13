package indexer

import (
	"io"
	"os"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
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

	Add(...*eagle.Entry) error
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
	fs      *fs.FS
}

func NewIndexer(fs *fs.FS, backend Backend) *Indexer {
	return &Indexer{
		fs:      fs,
		backend: backend,
	}
}

func (e *Indexer) Add(entries ...*eagle.Entry) error {
	return e.backend.Add(entries...)
}

func (e *Indexer) Remove(ids ...string) {
	e.backend.Remove(ids...)
}

func (e *Indexer) GetDrafts(opts *Pagination) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetDrafts(opts))
}

func (e *Indexer) GetUnlisted(opts *Pagination) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetUnlisted(opts))
}

func (e *Indexer) GetDeleted(opts *Pagination) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetDeleted(opts))
}

func (e *Indexer) GetSearch(opts *Query, query string) (eagle.Entries, error) {
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

func (e *Indexer) idsToEntries(ids []string, err error) (eagle.Entries, error) {
	if err != nil {
		return nil, err
	}

	entries := eagle.Entries{}

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
