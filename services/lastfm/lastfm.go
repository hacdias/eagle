package lastfm

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/media"
	"github.com/hacdias/eagle/pkg/lastfm"
)

const lastfmFileName = ".lastfm.json"

func getDailyListensID(year int, month time.Month, day int) string {
	return eagle.NewID("listened", time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

type LastFm struct {
	fs     *fs.FS
	media  *media.Media
	client *lastfm.LastFm
}

func NewLastFm(key, user string, fs *fs.FS, media *media.Media) *LastFm {
	return &LastFm{
		fs:     fs,
		media:  media,
		client: lastfm.NewLastFm(key, user),
	}
}

func (l *LastFm) FetchLastFmListens(year int, month time.Month, day int) (bool, error) {
	tracks, err := l.client.Fetch(year, month, day)
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
			} else if l.media != nil {
				url, err := l.media.UploadFromURL("media", t.OriginalImage, true)
				if err == nil {
					t.Image = url
					coverUploads[t.OriginalImage] = url
				}
			}
		}
	}

	filename := filepath.Join(fs.ContentDirectory, getDailyListensID(year, month, day), lastfmFileName)

	err = l.fs.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return false, err
	}
	return true, l.fs.WriteJSON(filename, tracks, fmt.Sprintf("lastfm data for %04d-%02d-%02d", year, month, day))
}

func (l *LastFm) CreateDailyListensEntry(year int, month time.Month, day int) error {
	id := eagle.NewID("listened", time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
	filename := filepath.Join(fs.ContentDirectory, id, lastfmFileName)
	tracks := []*lastfm.Track{}

	err := l.fs.ReadJSON(filename, &tracks)
	if err != nil {
		return err
	}

	stats := lastfm.ScrobblesStatistics(tracks)

	ee := &eagle.Entry{
		ID:      id,
		Content: "<!-- This post is automatically generated. -->\n\n",
		FrontMatter: eagle.FrontMatter{
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

	err = l.fs.SaveEntry(ee)
	if err != nil {
		return err
	}

	return nil
}

func (l *LastFm) DailyJob() error {
	today := time.Now().UTC()
	yesterday := today.AddDate(0, 0, -1)
	year, month, day := yesterday.Date()

	created, err := l.FetchLastFmListens(year, month, day)
	if err != nil {
		return err
	}

	if !created {
		return nil
	}

	return l.CreateDailyListensEntry(year, month, day)
}
