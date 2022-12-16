package hooks

import (
	"path/filepath"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/indexer"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hashicorp/go-multierror"
)

const (
	booksOverviewFile = "/content/books/overview"
)

type ReadsSummaryUpdater struct {
	fs      *fs.FS
	indexer *indexer.Indexer
}

func NewReadsSummaryUpdater(fs *fs.FS, indexer *indexer.Indexer) *ReadsSummaryUpdater {
	return &ReadsSummaryUpdater{
		fs:      fs,
		indexer: indexer,
	}
}

func (u *ReadsSummaryUpdater) EntryHook(_, e *eagle.Entry) error {
	if e.Helper().PostType() == mf2.TypeRead {
		return u.UpdateReadsSummary()
	}

	return nil
}

func (u *ReadsSummaryUpdater) UpdateReadsSummary() error {
	ee, err := u.indexer.GetBySection(&indexer.Query{}, "books")
	if err != nil {
		return err
	}

	toRead := eagle.Logs{}
	reading := eagle.Logs{}
	finished := eagle.Logs{}

	for _, e := range ee {
		mm := e.Helper()
		read := mm.Sub("read-of")
		statuses := mm.Properties.Objects("read-status")

		if read == nil || len(statuses) < 1 {
			continue
		}

		l := eagle.Log{
			URL:    e.ID,
			Name:   read.Name(),
			Author: read.String("author"),
			Date:   e.Published,
			Rating: mm.Int("rating"),
		}

		if e.Updated.After(e.Published) {
			l.Date = e.Updated
		}

		switch statuses[0].String("status") {
		case "to-read":
			toRead = append(toRead, l)
		case "reading":
			reading = append(reading, l)
		case "finished":
			finished = append(finished, l)
		}
	}

	return multierror.Append(
		nil,
		u.fs.WriteJSON(filepath.Join(booksOverviewFile, ".to-read.json"), toRead, "update toRead overview"),
		u.fs.WriteJSON(filepath.Join(booksOverviewFile, ".reading.json"), reading, "update reading overview"),
		u.fs.WriteJSON(filepath.Join(booksOverviewFile, ".read.json"), finished, "update read overview"),
	).ErrorOrNil()
}
