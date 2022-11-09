package hooks

import (
	"fmt"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/util"
)

type ReadsSummaryProvider interface {
	GetReadsSummary() (*entry.ReadsSummary, error)
}

// wip: ReadsSummary type should probably live with this package (make this a package?)

const (
	booksSummaryEntryID  = "/books/summary"
	booksSummaryTagStart = "<!--BOOKS-->"
	booksSummaryTagEnd   = "<!--/BOOKS-->"
)

type ReadsSummaryUpdater struct {
	Provider ReadsSummaryProvider
	Eagle    *eagle.Eagle // WIP: remove this once possible.
}

func (u *ReadsSummaryUpdater) EntryHook(e *entry.Entry, isNew bool) error {
	if e.Helper().PostType() == mf2.TypeRead {
		return u.UpdateReadsSummary()
	}

	return nil
}

func (u *ReadsSummaryUpdater) UpdateReadsSummary() error {
	stats, err := u.Provider.GetReadsSummary()
	if err != nil {
		return err
	}

	ee, err := u.Eagle.GetEntry(booksSummaryEntryID)
	if err != nil {
		return err
	}

	md := readsSummaryToMarkdown(stats)

	ee.Content, err = util.ReplaceInBetween(ee.Content, booksSummaryTagStart, booksSummaryTagEnd, md)
	if err != nil {
		return err
	}

	err = u.Eagle.SaveEntry(ee)
	if err != nil {
		return err
	}

	u.Eagle.RemoveCache(ee)
	return nil
}

func readsSummaryToMarkdown(stats *entry.ReadsSummary) string {
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

func readListToMarkdown(list entry.ReadList) string {
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
