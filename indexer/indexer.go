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

// TODO: make it GetEntries, GetDeletedEntries, etc
type Backend interface {
	io.Closer

	Remove(id string)
	Add(...*eagle.Entry) error

	GetTaxonomyTerms(taxonomy string) (eagle.Terms, error)
	Search(opt *Query, search *Search) ([]string, error)

	GetAll(opts *Query) ([]string, error)
	GetDeleted(opts *Pagination) ([]string, error)
	GetDrafts(opts *Pagination) ([]string, error)

	ByTaxonomy(opt *Query, taxonomy, term string) ([]string, error)
	BySection(opt *Query, section string) ([]string, error)
	ByDate(opts *Query, year, month, day int) ([]string, error)

	Count() (int, error)
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

func (e *Indexer) Remove(id string) {
	e.backend.Remove(id)
}

func (e *Indexer) Add(entries ...*eagle.Entry) error {
	return e.backend.Add(entries...)
}

func (e *Indexer) GetTaxonomyTerms(taxonomy string) (eagle.Terms, error) {
	return e.backend.GetTaxonomyTerms(taxonomy)
}

func (e *Indexer) Search(opts *Query, search *Search) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.backend.Search(opts, search))
}

func (e *Indexer) GetAll(opts *Query) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.backend.GetAll(opts))
}

func (e *Indexer) GetByTaxonomy(opts *Query, taxonomy, term string) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.backend.ByTaxonomy(opts, taxonomy, term))
}

func (e *Indexer) GetBySection(opts *Query, section string) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.backend.BySection(opts, section))
}

func (e *Indexer) GetByDate(opts *Query, year, month, day int) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.backend.ByDate(opts, year, month, day))
}

func (e *Indexer) GetDeleted(opts *Pagination) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.backend.GetDeleted(opts))
}

func (e *Indexer) GetDrafts(opts *Pagination) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.backend.GetDrafts(opts))
}

func (e *Indexer) Count() (int, error) {
	return e.backend.Count()
}

func (e *Indexer) Close() error {
	return e.backend.Close()
}

func (e *Indexer) idsToEntries(ids []string, err error) ([]*eagle.Entry, error) {
	if err != nil {
		return nil, err
	}

	entries := []*eagle.Entry{}

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
