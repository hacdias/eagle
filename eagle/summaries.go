package eagle

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hacdias/eagle/v4/entry"
)

const (
	booksSummaryEntryID  = "/books/summary"
	booksSummaryTagStart = "<!--BOOKS-->"
	booksSummaryTagEnd   = "<!--/BOOKS-->"
)

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

	ee.Content, err = replaceBetween(ee.Content, booksSummaryTagStart, booksSummaryTagEnd, md)
	if err != nil {
		return err
	}

	err = e.saveEntry(ee)
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
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

const (
	watchesSummaryEntryID  = "/watches/summary"
	watchesSummaryTagStart = "<!--WATCHES-->"
	watchesSummaryTagEnd   = "<!--/WATCHES-->"
)

func (e *Eagle) UpdateWatchesSummary() error {
	stats, err := e.db.WatchesSummary()
	if err != nil {
		return err
	}

	ee, err := e.GetEntry(watchesSummaryEntryID)
	if err != nil {
		return err
	}

	md := watchesSummaryToMarkdown(stats)

	ee.Content, err = replaceBetween(ee.Content, watchesSummaryTagStart, watchesSummaryTagEnd, md)
	if err != nil {
		return err
	}

	err = e.saveEntry(ee)
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}

func watchesSummaryToMarkdown(stats *entry.WatchesSummary) string {
	summary := "## 📺 Series {#series}\n\n"
	summary += "<div class='box'>\n\n"
	summary += watchListToMarkdown(stats.Series)
	summary += "\n</div>\n\n## 🎬 Movies {#movies}\n\n<div class='box'>\n\n"
	summary += watchListToMarkdown(stats.Movies)
	summary += "\n</div>"
	return summary
}

func watchListToMarkdown(list []*entry.Watch) string {
	md := ""

	for _, watch := range list {
		md += fmt.Sprintf("- [%s](%s) <small>last watched in %s</small>\n", watch.Name, watch.ID, watch.Date.Format("January 2006"))
	}

	return md
}

func replaceBetween(s, start, end, new string) (string, error) {
	startIdx := strings.Index(s, start)
	endIdx := strings.LastIndex(s, end)

	if startIdx == -1 || endIdx == -1 {
		return "", errors.New("start tag or end tag not present")
	}

	return s[0:startIdx] +
		start + "\n" + new + "\n" + end +
		s[endIdx+len(end):], nil
}
