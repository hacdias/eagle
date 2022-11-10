package hooks

import (
	"fmt"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/util"
)

// WIP: perhaps this should live in a separate package and include ReadsSummary there.

type ReadsSummaryProvider interface {
	GetReadsSummary() (*eagle.ReadsSummary, error)
}

const (
	booksSummaryEntryID  = "/books/summary"
	booksSummaryTagStart = "<!--BOOKS-->"
	booksSummaryTagEnd   = "<!--/BOOKS-->"
)

type ReadsSummaryUpdater struct {
	fs       *fs.FS
	provider ReadsSummaryProvider
}

func NewReadsSummaryUpdater(fs *fs.FS, provider ReadsSummaryProvider) *ReadsSummaryUpdater {
	return &ReadsSummaryUpdater{
		fs:       fs,
		provider: provider,
	}
}

func (u *ReadsSummaryUpdater) EntryHook(e *eagle.Entry, isNew bool) error {
	if e.Helper().PostType() == mf2.TypeRead {
		return u.UpdateReadsSummary()
	}

	return nil
}

func (u *ReadsSummaryUpdater) UpdateReadsSummary() error {
	stats, err := u.provider.GetReadsSummary()
	if err != nil {
		return err
	}

	_, err = u.fs.TransformEntry(booksSummaryEntryID, func(e *eagle.Entry) (*eagle.Entry, error) {
		var err error
		md := readsSummaryToMarkdown(stats)
		e.Content, err = util.ReplaceInBetween(e.Content, booksSummaryTagStart, booksSummaryTagEnd, md)
		return e, err
	})
	return err
}

func readsSummaryToMarkdown(stats *eagle.ReadsSummary) string {
	summary := "## 📖 Reading {#reading}\n\n"

	if len(stats.Reading) == 0 {
		summary += "Not reading any books at the moment.\n"
	} else {
		summary += readListToMarkdown(stats.Reading)
	}

	summary += "\n## 📚 To Read {#to-read}\n\n"

	if len(stats.ToRead) == 0 {
		summary += "Not books on the queue at the moment.\n"
	} else {
		summary += readListToMarkdown(stats.ToRead)
	}

	summary += "\n## 📕 Finished {#finished}\n\n"

	for _, year := range stats.Finished.Years {
		books := stats.Finished.Map[year]

		if year == 1 {
			summary += fmt.Sprintf("\n### Others <small>(%d books)</small> {#others}\n\n", len(books))
		} else {
			summary += fmt.Sprintf("\n### %d <small>(%d books)</small> {#%d}\n\n", year, len(books), year)
		}

		summary += readListToMarkdown(books)
	}

	return summary
}

func readListToMarkdown(list eagle.ReadList) string {
	md := ""

	list.SortByName()
	for _, book := range list {
		md += fmt.Sprintf("- [%s](%s)", book.Name, book.ID)
		if book.Author != "" {
			md += fmt.Sprintf(" <small>by %s</small>", book.Author)
		}
		md += "\n"
	}

	return md
}
