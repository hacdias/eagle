package database

import (
	"os"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
)

type DatabaseWrapper struct {
	DB Database
	FS *fs.FS
}

func (e *DatabaseWrapper) Remove(id string) {
	e.DB.Remove(id)
}

func (e *DatabaseWrapper) Add(entries ...*eagle.Entry) error {
	return e.DB.Add(entries...)
}

func (e *DatabaseWrapper) GetTags() ([]string, error) {
	return e.DB.GetTags()
}

func (e *DatabaseWrapper) GetEmojis() ([]string, error) {
	return e.DB.GetEmojis()
}

func (e *DatabaseWrapper) Search(opts *QueryOptions, search *SearchOptions) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.Search(opts, search))
}

func (e *DatabaseWrapper) GetAll(opts *QueryOptions) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.GetAll(opts))
}

func (e *DatabaseWrapper) GetByTag(opts *QueryOptions, tag string) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.ByTag(opts, tag))
}

func (e *DatabaseWrapper) GetByEmoji(opts *QueryOptions, emoji string) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.ByEmoji(opts, emoji))
}

func (e *DatabaseWrapper) GetBySection(opts *QueryOptions, sections ...string) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.BySection(opts, sections...))
}

func (e *DatabaseWrapper) GetByDate(opts *QueryOptions, year, month, day int) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.ByDate(opts, year, month, day))
}

func (e *DatabaseWrapper) GetDeleted(opts *PaginationOptions) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.GetDeleted(opts))
}

func (e *DatabaseWrapper) GetDrafts(opts *PaginationOptions) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.GetDrafts(opts))
}

func (e *DatabaseWrapper) GetUnlisted(opts *PaginationOptions) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.GetUnlisted(opts))
}

func (e *DatabaseWrapper) GetPrivate(opts *PaginationOptions, audience string) ([]*eagle.Entry, error) {
	return e.idsToEntries(e.DB.GetPrivate(opts, audience))
}

func (e *DatabaseWrapper) idsToEntries(ids []string, err error) ([]*eagle.Entry, error) {
	if err != nil {
		return nil, err
	}

	entries := []*eagle.Entry{}

	for _, id := range ids {
		entry, err := e.FS.GetEntry(id)
		if err != nil {
			if os.IsNotExist(err) {
				e.DB.Remove(id)
			} else {
				return nil, err
			}
		} else {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// wip: server
// func (e *DatabaseWrapper) indexAll() {
// 	entries, err := e.FS.GetEntries(false)
// 	if err != nil {
// 		e.Notifier.Error(err)
// 		return
// 	}

// 	start := time.Now()
// 	err = e.DB.Add(entries...)
// 	if err != nil {
// 		e.Notifier.Error(err)
// 	}
// 	e.log.Infof("database update took %dms", time.Since(start).Milliseconds())
// }
