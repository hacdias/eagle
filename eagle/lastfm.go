package eagle

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/pkg/lastfm"
)

const lastfmFileName = ".lastfm.json"

func getDailyListensID(year int, month time.Month, day int) string {
	return entry.NewID("listened", time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

func (e *Eagle) FetchLastFmListens(year int, month time.Month, day int) (bool, error) {
	if e.lastfm == nil {
		return false, errors.New("lastfm is not implemented")
	}

	tracks, err := e.lastfm.Fetch(year, month, day)
	if err != nil {
		return false, err
	}

	if len(tracks) == 0 {
		return false, nil
	}

	coverUploads := map[string]string{}

	for _, t := range tracks {
		if t.Image == "" && t.OriginalImage != "" {
			if dst, ok := coverUploads[t.OriginalImage]; ok {
				t.Image = dst
			} else {
				url, err := e.uploadFromURL("media", t.OriginalImage, true)
				if err == nil {
					t.Image = url
					coverUploads[t.OriginalImage] = url
				}
			}
		}
	}

	filename := filepath.Join(ContentDirectory, getDailyListensID(year, month, day), lastfmFileName)

	err = e.fs.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return false, err
	}
	return true, e.fs.WriteJSON(filename, tracks, fmt.Sprintf("lastfm data for %04d-%02d-%02d", year, month, day))
}

func (e *Eagle) CreateDailyListensEntry(year int, month time.Month, day int) error {
	id := entry.NewID("listened", time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
	filename := filepath.Join(ContentDirectory, id, lastfmFileName)
	tracks := []*lastfm.Track{}

	err := e.fs.ReadJSON(filename, &tracks)
	if err != nil {
		return err
	}

	stats := lastfm.ScrobblesStatistics(tracks)

	ee := &entry.Entry{
		ID:      id,
		Content: "<!-- This post is automatically generated. -->\n\n",
		FrontMatter: entry.FrontMatter{
			Sections:    []string{"listens"},
			Description: fmt.Sprintf("Listened to %d tracks from %d artists across %d albums", stats.TotalTracks, stats.TotalArtists, stats.TotalAlbums),
			Properties: map[string]interface{}{
				"scrobbles-count": stats.TotalScrobbles,
				"artists-count":   stats.TotalArtists,
				"tracks-count":    stats.TotalTracks,
				"albums-count":    stats.TotalAlbums,
				"total-duration":  stats.TotalDuration.Hours(),
				"listen-of":       "summary",
			},
			Published: time.Date(year, month, day, 23, 59, 59, 0, time.UTC),
		},
	}

	if len(stats.Albums) > 0 {
		imgSrc := stats.Albums[0].Track.Image
		if imgSrc != "" {
			ee.Content += fmt.Sprintf("![%s](%s?class=right+album)\n\n", stats.Albums[0].Track.Album.Name, imgSrc)
		}
	}

	ee.Content += fmt.Sprintf(
		"Today, I scrobbled **%d** times to **%d** unique tracks from **%d** different artists across **%d** different albums.\n",
		stats.TotalScrobbles, stats.TotalTracks, stats.TotalArtists, stats.TotalAlbums,
	)

	ee.Content += fmt.Sprintf(
		"I listened to approximately **%.2f** hours of music.\n",
		stats.TotalDuration.Hours(),
	)

	if len(stats.Tracks)+len(stats.Albums)+len(stats.Artists) > 0 {
		ee.Content += "Today's tops are:\n"

		if len(stats.Tracks) > 0 {
			ee.Content += fmt.Sprintf(
				"- 🎶 **Track**: %s <small>[%d scrobbles]</small>\n",
				stats.Tracks[0].Name, stats.Tracks[0].Scrobbles,
			)
		}

		if len(stats.Albums) > 0 {
			ee.Content += fmt.Sprintf(
				"- 💿 **Album**: %s <small>[%d scrobbles]</small>\n",
				stats.Albums[0].Name, stats.Albums[0].Scrobbles,
			)
		}

		if len(stats.Artists) > 0 {
			ee.Content += fmt.Sprintf(
				"- 🎤 **Artist**: %s <small>[%d scrobbles]</small>\n\n",
				stats.Artists[0].Name, stats.Artists[0].Scrobbles,
			)
		}
	}

	ee.Content += "<details class=scrobbles-log>\n<summary>Scrobbles Log</summary>\n\n"

	for _, t := range tracks {
		img := "💿"
		if t.Image != "" {
			img = fmt.Sprintf("<img src=%s class=emoji />", t.Image)
		}

		ee.Content += fmt.Sprintf(
			"- <time class=tab>%s</time> %s %s <small>by %s</small>\n",
			t.Date.UTC().Format("15:04"), img, t.Name, t.Artist.Name,
		)
	}

	ee.Content += "\n\n</details>\n"

	err = e.SaveEntry(ee)
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}

func (e *Eagle) initScrobbleCron() error {
	_, err := e.cron.AddFunc("CRON_TZ=UTC 00 05 * * *", func() {
		today := time.Now().UTC()
		yesterday := today.AddDate(0, 0, -1)
		year, month, day := yesterday.Date()

		created, err := e.FetchLastFmListens(year, month, day)
		if err != nil {
			e.Notifier.Error(fmt.Errorf("daily scrobbles cron job: %w", err))
			return
		}

		if !created {
			return
		}

		err = e.CreateDailyListensEntry(year, month, day)
		if err != nil {
			e.Notifier.Error(fmt.Errorf("daily scrobbles cron job: %w", err))
			return
		}
	})

	return err
}
