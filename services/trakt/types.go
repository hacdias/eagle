package trakt

import (
	"fmt"
	"sort"
	"time"
)

// https://trakt.docs.apiary.io/#introduction/standard-media-objects

type traktGeneric struct {
	Title string   `json:"title"`
	Year  int      `json:"year"`
	IDs   traktIDs `json:"ids"`
}

type traktMovie traktGeneric

type traktShow traktGeneric

type traktEpisode struct {
	Title  string   `json:"title"`
	Season int      `json:"season"`
	Number int      `json:"number"`
	IDs    traktIDs `json:"ids"`
}

type traktIDs struct {
	Trakt int    `json:"trakt"`
	IMDb  string `json:"imdb"`
	TMDb  int    `json:"tmdb"`
	Slug  string `json:"slug,omitempty"`
	TVDb  int    `json:"tvdb,omitempty"`
}

type traktHistoryItem struct {
	ID        int64         `json:"id"`
	WatchedAt time.Time     `json:"watched_at"`
	Action    string        `json:"action"`
	Type      string        `json:"type"`
	Movie     *traktMovie   `json:"movie,omitempty"`
	Episode   *traktEpisode `json:"episode,omitempty"`
	Show      *traktShow    `json:"show,omitempty"`
}

type traktHistory []traktHistoryItem

func (h traktHistory) sort() {
	sort.SliceStable(h, func(i, j int) bool {
		return h[i].WatchedAt.After(h[j].WatchedAt)
	})
}

type summary struct {
	UniqueMovies  int          `json:"uniqueMovies"`
	UniqueShows   int          `json:"uniqueShows"`
	TotalMovies   int          `json:"totalMovies"`
	TotalEpisodes int          `json:"totalEpisodes"`
	TotalWatches  int          `json:"totalWatches"`
	Movies        traktHistory `json:"movies"`
	Shows         traktHistory `json:"shows"`
}

func (s *summary) toMarkdown() string {
	summary := "## 📺 Series {#series}\n\n"
	summary += "<div class='box'>\n\n"
	summary += showsToMarkdown(s.Shows)
	summary += "\n</div>\n\n## 🎬 Movies {#movies}\n\n<div class='box'>\n\n"
	summary += moviesToMarkdown(s.Movies)
	summary += "\n</div>"
	return summary
}

func showsToMarkdown(shows traktHistory) string {
	md := ""

	for _, s := range shows {
		md += fmt.Sprintf(
			"- %s <small>last watched in %s</small>\n",
			s.Show.Title, s.WatchedAt.Format("January 2006"),
		)
	}

	return md
}

func moviesToMarkdown(movies traktHistory) string {
	md := ""

	for _, m := range movies {
		md += fmt.Sprintf(
			"- %s <small>last watched in %s</small>\n",
			m.Movie.Title, m.WatchedAt.Format("January 2006"),
		)
	}

	return md
}
