package eagle

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hacdias/eagle/v4/entry"
)

const (
	booksSummaryEntryID  = "/books/summary"
	booksSummaryTagStart = "<!--BOOKS-->"
	booksSummaryTagEnd   = "<!--/BOOKS-->"
)

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
			summary += fmt.Sprintf("### Others <small>(%d books)</small> {#others}\n\n", len(books))
		} else {
			summary += fmt.Sprintf("### %d <small>(%d books)</small> {#%d}\n\n", year, len(books), year)
		}

		summary += readListToMarkdown(books)
	}

	return summary
}

func (e *Eagle) UpdateReadsSummary() error {
	stats, err := e.db.ReadsSummary()
	if err != nil {
		return err
	}

	ee, err := e.GetEntry(booksSummaryEntryID)
	if err != nil {
		return err
	}

	md := readsSummaryToMarkdown(stats)

	startIdx := strings.Index(ee.Content, booksSummaryTagStart)
	endIdx := strings.LastIndex(ee.Content, booksSummaryTagEnd)

	if startIdx == -1 || endIdx == -1 {
		return errors.New("book summary tags not present")
	}

	ee.Content = ee.Content[0:startIdx] +
		booksSummaryTagStart + "\n" + md + booksSummaryTagEnd + "\n" +
		ee.Content[endIdx+len(booksSummaryTagEnd):]

	return e.saveEntry(ee)
}

const WatchesSummary = "_summary.watches.json"

func (e *Eagle) UpdateWatchesSummary() error {
	stats, err := e.db.WatchesSummary()
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, WatchesSummary)
	err = e.fs.WriteJSON(filename, stats, "update watches summary")
	if err != nil {
		return err
	}

	ee, err := e.GetEntry("/watches/summary")
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}
