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
	Pagination     *Pagination
	OrderByUpdated bool

	WithDrafts  bool
	WithDeleted bool

	// Empty matches all visibilities.
	Visibility []eagle.Visibility

	// Empty matches all audiences.
	Audience string
}

type Search struct {
	Query    string
	Sections []string
}

type Backend interface {
	io.Closer

	Add(...*eagle.Entry) error
	Remove(ids ...string)

	GetAll(opts *Query) ([]string, error)
	GetDrafts(opts *Pagination) ([]string, error)
	GetDeleted(opts *Pagination) ([]string, error)

	GetBySection(opt *Query, section string) ([]string, error)
	GetByTaxonomy(opt *Query, taxonomy, term string) ([]string, error)
	GetByDate(opts *Query, year, month, day int) ([]string, error)

	GetTaxonomyTerms(taxonomy string) (eagle.Terms, error)
	GetSearch(opt *Query, search *Search) ([]string, error)
	GetCount() (int, error)
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

func (e *Indexer) GetAll(opts *Query) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetAll(opts))
}

func (e *Indexer) GetDrafts(opts *Pagination) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetDrafts(opts))
}

func (e *Indexer) GetDeleted(opts *Pagination) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetDeleted(opts))
}

func (e *Indexer) GetBySection(opts *Query, section string) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetBySection(opts, section))
}

func (e *Indexer) GetByTaxonomy(opts *Query, taxonomy, term string) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetByTaxonomy(opts, taxonomy, term))
}

func (e *Indexer) GetByDate(opts *Query, year, month, day int) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetByDate(opts, year, month, day))
}

func (e *Indexer) GetTaxonomyTerms(taxonomy string) (eagle.Terms, error) {
	return e.backend.GetTaxonomyTerms(taxonomy)
}

func (e *Indexer) GetSearch(opts *Query, search *Search) (eagle.Entries, error) {
	return e.idsToEntries(e.backend.GetSearch(opts, search))
}

func (e *Indexer) GetCount() (int, error) {
	return e.backend.GetCount()
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
