package hooks

import (
	"fmt"

	"github.com/hacdias/eagle/v4/eagle"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/hacdias/eagle/v4/util"
)

// wip: WatchesSummary type should probably live with this package (make this a package?)

type WatchesSummaryProvider interface {
	GetWatchesSummary() (*entry.WatchesSummary, error)
}

type WatchesSummaryUpdater struct {
	Provider WatchesSummaryProvider
	Eagle    *eagle.Eagle // WIP: remove this once possible.
}

const (
	watchesSummaryEntryID  = "/watches/summary"
	watchesSummaryTagStart = "<!--WATCHES-->"
	watchesSummaryTagEnd   = "<!--/WATCHES-->"
)

func (u *WatchesSummaryUpdater) EntryHook(e *entry.Entry, isNew bool) error {
	if e.Helper().PostType() == mf2.TypeWatch {
		return u.UpdateWatchesSummary()
	}
	return nil
}

func (u *WatchesSummaryUpdater) UpdateWatchesSummary() error {
	stats, err := u.Provider.GetWatchesSummary()
	if err != nil {
		return err
	}

	_, err = u.Eagle.TransformEntry(watchesSummaryEntryID, func(e *entry.Entry) (*entry.Entry, error) {
		var err error
		md := watchesSummaryToMarkdown(stats)
		e.Content, err = util.ReplaceInBetween(e.Content, watchesSummaryTagStart, watchesSummaryTagEnd, md)
		return e, err
	})
	return err
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
