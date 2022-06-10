package eagle

import (
	"os"
	"time"

	"github.com/hacdias/eagle/v4/database"
	"github.com/hacdias/eagle/v4/entry"
)

func (e *Eagle) GetTags() ([]string, error) {
	return e.db.GetTags()
}

func (e *Eagle) GetEmojis() ([]string, error) {
	return e.db.GetEmojis()
}

func (e *Eagle) Search(opts *database.QueryOptions, query string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.Search(opts, query))
}

func (e *Eagle) GetAll(opts *database.QueryOptions) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.GetAll(opts))
}

func (e *Eagle) GetByTag(opts *database.QueryOptions, tag string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.ByTag(opts, tag))
}

func (e *Eagle) GetByEmoji(opts *database.QueryOptions, emoji string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.ByEmoji(opts, emoji))
}

func (e *Eagle) GetBySection(opts *database.QueryOptions, sections ...string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.BySection(opts, sections...))
}

func (e *Eagle) GetByDate(opts *database.QueryOptions, year, month, day int) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.ByDate(opts, year, month, day))
}

func (e *Eagle) GetByProperty(opts *database.QueryOptions, property, value string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.ByProperty(opts, property, value))
}

func (e *Eagle) GetDeleted(opts *database.PaginationOptions) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.GetDeleted(opts))
}

func (e *Eagle) GetDrafts(opts *database.PaginationOptions) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.GetDrafts(opts))
}

func (e *Eagle) GetUnlisted(opts *database.PaginationOptions) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.GetUnlisted(opts))
}

func (e *Eagle) GetPrivate(opts *database.PaginationOptions, audience string) ([]*entry.Entry, error) {
	return e.idsToEntries(e.db.GetPrivate(opts, audience))
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
