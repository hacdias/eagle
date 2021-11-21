package eagle

import (
	"os"
	"time"

	"github.com/hacdias/eagle/v2/database"
	"github.com/hacdias/eagle/v2/entry"
)

func (e *Eagle) GetTags() ([]string, error) {
	return e.db.GetTags()
}

func (e *Eagle) Search(opts *database.QueryOptions, query string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.Search(opts, query))
}

func (e *Eagle) ByTag(opts *database.QueryOptions, tag string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.ByTag(opts, tag))
}

func (e *Eagle) BySection(opts *database.QueryOptions, sections ...string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.BySection(opts, sections...))
}

func (e *Eagle) ByDate(opts *database.QueryOptions, year, month, day int) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.ByDate(opts, year, month, day))
}

func (e *Eagle) GetDeleted(opts *database.PaginationOptions) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.GetDeleted(opts))
}

func (e *Eagle) GetDrafts(opts *database.PaginationOptions) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.GetDrafts(opts))
}

func (e *Eagle) idsToEntries(ids []string, err error) ([]*entry.Entry, error) {
	if err != nil {
		return nil, err
	}

	entries := []*entry.Entry{}

	for _, id := range ids {
		entry, err := e.GetEntry(id)
		if err != nil {
			if os.IsNotExist(err) {
				e.db.Remove(id)
			} else {
				return nil, err
			}
		} else {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (e *Eagle) indexAll() {
	entries, err := e.GetEntries(false)
	if err != nil {
		e.Notifier.Error(err)
		return
	}

	start := time.Now()
	err = e.db.Add(entries...)
	if err != nil {
		e.Notifier.Error(err)
	}
	e.log.Infof("database update took %dms", time.Since(start).Milliseconds())
}
