package hooks

import (
	"fmt"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/fs"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/util"
)

// WIP: perhaps this should live in a separate package and include ReadsSummary there.

type WatchesSummaryProvider interface {
	GetWatchesSummary() (*eagle.WatchesSummary, error)
}

type WatchesSummaryUpdater struct {
	fs       *fs.FS
	provider WatchesSummaryProvider
}

func NewWatchesSummaryUpdater(fs *fs.FS, provider WatchesSummaryProvider) *WatchesSummaryUpdater {
	return &WatchesSummaryUpdater{
		fs:       fs,
		provider: provider,
	}
}

const (
	watchesSummaryEntryID  = "/watches/summary"
	watchesSummaryTagStart = "<!--WATCHES-->"
	watchesSummaryTagEnd   = "<!--/WATCHES-->"
)

func (u *WatchesSummaryUpdater) EntryHook(e *eagle.Entry, isNew bool) error {
	if e.Helper().PostType() == mf2.TypeWatch {
		return u.UpdateWatchesSummary()
	}
	return nil
}

func (u *WatchesSummaryUpdater) UpdateWatchesSummary() error {
	stats, err := u.provider.GetWatchesSummary()
	if err != nil {
		return err
	}

	_, err = u.fs.TransformEntry(watchesSummaryEntryID, func(e *eagle.Entry) (*eagle.Entry, error) {
		var err error
		md := watchesSummaryToMarkdown(stats)
		e.Content, err = util.ReplaceInBetween(e.Content, watchesSummaryTagStart, watchesSummaryTagEnd, md)
		return e, err
	})
	return err
}

func watchesSummaryToMarkdown(stats *eagle.WatchesSummary) string {
	summary := "## 📺 Series {#series}\n\n"
	summary += "<div class='box'>\n\n"
	summary += watchListToMarkdown(stats.Series)
	summary += "\n</div>\n\n## 🎬 Movies {#movies}\n\n<div class='box'>\n\n"
	summary += watchListToMarkdown(stats.Movies)
	summary += "\n</div>"
	return summary
}

func watchListToMarkdown(list []*eagle.Watch) string {
	md := ""

	for _, watch := range list {
		md += fmt.Sprintf("- [%s](%s) <small>last watched in %s</small>\n", watch.Name, watch.ID, watch.Date.Format("January 2006"))
	}

	return md
}
