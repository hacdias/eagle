package eagle

import (
	"fmt"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/util"
)

const (
	watchesSummaryEntryID  = "/watches/summary"
	watchesSummaryTagStart = "<!--WATCHES-->"
	watchesSummaryTagEnd   = "<!--/WATCHES-->"
)

func (e *Eagle) UpdateWatchesSummary() error {
	stats, err := e.DB.WatchesSummary()
	if err != nil {
		return err
	}

	ee, err := e.GetEntry(watchesSummaryEntryID)
	if err != nil {
		return err
	}

	md := watchesSummaryToMarkdown(stats)

	ee.Content, err = util.ReplaceInBetween(ee.Content, watchesSummaryTagStart, watchesSummaryTagEnd, md)
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
