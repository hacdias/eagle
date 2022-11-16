package hooks

import (
	"fmt"
	"sort"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/hacdias/eagle/util"
)

// TODO: perhaps extract this to a separate package.

type Read struct {
	ID     string    `json:"id"`
	Date   time.Time `json:"date"`
	Name   string    `json:"name"`
	Author string    `json:"author"`
}

type ReadList []*Read

type ReadsSummary struct {
	ToRead   ReadList    `json:"to-read"`
	Reading  ReadList    `json:"reading"`
	Finished ReadsByYear `json:"finished"`
}

type ReadsByYear struct {
	Years []int
	Map   map[int]ReadList
}

func (rd ReadList) ByYear() *ReadsByYear {
	years := []int{}
	byYear := map[int]ReadList{}

	for _, r := range rd {
		year := r.Date.Year()

		_, ok := byYear[year]
		if !ok {
			years = append(years, year)
			byYear[year] = ReadList{}
		}

		byYear[year] = append(byYear[year], r)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	for _, year := range years {
		byYear[year].SortByName()
	}

	return &ReadsByYear{
		Years: years,
		Map:   byYear,
	}
}

func (rd ReadList) SortByName() {
	sort.SliceStable(rd, func(i, j int) bool {
		return rd[i].Name < rd[j].Name
	})
}

type ReadsSummaryProvider interface {
	GetReadsSummary() (*ReadsSummary, error)
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

func (u *ReadsSummaryUpdater) EntryHook(_, e *eagle.Entry) error {
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

func readsSummaryToMarkdown(stats *ReadsSummary) string {
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

func readListToMarkdown(list ReadList) string {
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
