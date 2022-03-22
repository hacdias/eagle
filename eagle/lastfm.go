package eagle

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/entry"
	"github.com/hacdias/eagle/v3/entry/mf2"
	"github.com/hacdias/eagle/v3/log"
)

type Lastfm struct {
	*config.Lastfm
}

func (l *Lastfm) Fetch(year int, month time.Month, day int) ([]*lastfmTrack, error) {
	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	from := midnight.Unix()
	to := midnight.Unix() + 86400 // +1 Day

	tracks := []*lastfmTrack{}
	page := 1

	for {
		res, err := l.recentTracks(page, from, to)
		if err != nil {
			return nil, err
		}

		for _, track := range res.Tracks {
			if track.NowPlaying {
				continue
			}

			info, err := l.trackInfo(track)
			if err == nil {
				track.Duration = time.Duration(info.Duration) * time.Millisecond

				for _, tag := range info.Tags.Tags {
					track.Tags = append(track.Tags, strings.ToLower(tag.Name))
				}
			} else {
				log.S().Errorf("could not download track info: %s", err)
			} // When this fails, we assume an average time of 3m30s.

			tracks = append(tracks, track)
		}

		if res.Page < res.TotalPages {
			page++
		} else {
			break
		}
	}

	return tracks, nil
}

func (l *Lastfm) recentTracks(page int, from, to int64) (*lastfmTracks, error) {
	limit := 200
	u, err := url.Parse("https://ws.audioscrobbler.com/2.0/")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("method", "user.getrecenttracks")
	q.Set("user", l.User)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page))
	q.Set("api_key", l.Key)

	if from != 0 {
		q.Set("from", strconv.FormatInt(from, 10))
	}

	if to != 0 {
		q.Set("to", strconv.FormatInt(to, 10))
	}

	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response *lastfmTracksResponse
	err = xml.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.RecentTracks, nil
}

func (l *Lastfm) trackInfo(t *lastfmTrack) (*lastfmTrackInfo, error) {
	u, err := url.Parse("https://ws.audioscrobbler.com/2.0/")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("method", "track.getInfo")
	q.Set("api_key", l.Key)

	q.Set("track", t.Name)
	q.Set("artist", t.Artist.Name)

	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response *lastfmTrackInfoResponse
	err = xml.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Track, nil
}

func dailyScrobblesID(year int, month time.Month, day int) string {
	return entry.NewID("daily-scrobbles", time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

func (e *Eagle) FetchLastfmScrobbles(year int, month time.Month, day int) error {
	if e.lastfm == nil {
		return errors.New("lastfm is not implemented")
	}

	scrobbles, err := e.lastfm.Fetch(year, month, day)
	if err != nil {
		return err
	}

	if len(scrobbles) == 0 {
		return nil
	}

	artists := map[string]bool{}
	tracks := map[string]bool{}
	totalDuration := time.Duration(0)
	listenOf := []map[string]interface{}{}

	for _, scrobble := range scrobbles {
		key := scrobble.Name + scrobble.Artist.Name
		if _, ok := tracks[key]; !ok {
			tracks[key] = true
		}

		key = scrobble.Artist.Name
		if _, ok := artists[key]; !ok {
			artists[key] = true
		}

		totalDuration += scrobble.DurationOrAverage()
		listenOf = append(listenOf, scrobble.toFlatMF2())
	}

	id := dailyScrobblesID(year, month, day)

	ee := &entry.Entry{
		ID: id,
		Frontmatter: entry.Frontmatter{
			Sections:    []string{"listens"},
			Description: fmt.Sprintf("Listened to %d tracks for %.2f hours", len(tracks), totalDuration.Hours()),
			Properties: map[string]interface{}{
				"scrobbles-count": len(scrobbles),
				"artists-count":   len(artists),
				"tracks-count":    len(tracks),
				"total-duration":  totalDuration.Hours(),
				"listen-of":       listenOf,
				"category":        []string{"daily-scrobbles"},
			},
			Published: time.Date(year, month, day, 23, 59, 59, 0, time.UTC),
		},
	}

	err = e.SaveEntry(ee)
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}

func weekdayToIndex(d time.Weekday) int {
	switch d {
	case time.Monday:
		return 0
	case time.Tuesday:
		return 1
	case time.Wednesday:
		return 2
	case time.Thursday:
		return 3
	case time.Friday:
		return 4
	case time.Saturday:
		return 5
	case time.Sunday:
		return 6
	}

	panic("non existing week day")
}

// end is not included
func (e *Eagle) getScrobblesBetweenDates(start, end time.Time) []*mf2.FlatHelper {
	scrobbles := []*mf2.FlatHelper{}
	cur := start

	for !cur.Equal(end) {
		id := dailyScrobblesID(cur.Date())

		ee, err := e.GetEntry(id)
		if err == nil {
			mm := ee.Helper()

			if mm.PostType() == mf2.TypeListen {
				subs := mm.Subs(mm.TypeProperty())
				scrobbles = append(scrobbles, subs...)
			}
		}

		cur = cur.AddDate(0, 0, 1)
	}

	return scrobbles
}

func (e *Eagle) MakeMonthlyScrobblesReport(year int, month time.Month) error {
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	scrobbles := e.getScrobblesBetweenDates(start, end)
	if len(scrobbles) == 0 {
		return nil
	}

	stats, err := statsFromScrobbles(scrobbles)
	if err != nil {
		return err
	}

	// Publish as if it's the last second of the week.
	published := end.Add(-time.Second)

	stats.Start = start
	stats.End = published

	days := end.Sub(start).Hours() / 24

	stats.TracksPerDay = stats.Scrobbles / int(days)
	stats.DurationPerDay = stats.TotalDuration / int(days)

	id := entry.NewID("monthly-scrobbles", published)
	ee := &entry.Entry{
		ID: id,
		Frontmatter: entry.Frontmatter{
			Title: fmt.Sprintf("%s in Music", published.Month().String()),
			Description: fmt.Sprintf(
				"In %s %d, I listened to %d tracks from %d different artists for a total of %.2f hours.",
				published.Month().String(),
				published.Year(),
				stats.UniqueTracks,
				stats.UniqueArtists,
				durationFromSeconds(float64(stats.TotalDuration)).Hours(),
			),
			Template: "scrobbles-report",
			Sections: []string{"listens"},
			Properties: map[string]interface{}{
				"category": []string{"monthly-scrobbles"},
			},
			Published: published,
		},
	}

	err = e.SaveEntry(ee)
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, id, "_stats.json")
	err = e.fs.WriteJSON(filename, stats, "update stats")
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}

func (e *Eagle) MakeYearlyScrobblesReport(year int) error {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	scrobbles := e.getScrobblesBetweenDates(start, end)
	if len(scrobbles) == 0 {
		return nil
	}

	stats, err := statsFromScrobbles(scrobbles)
	if err != nil {
		return err
	}

	// Publish as if it's the last second of the week.
	published := end.Add(-time.Second)

	stats.Start = start
	stats.End = published

	days := end.Sub(start).Hours() / 24

	stats.TracksPerDay = stats.Scrobbles / int(days)
	stats.DurationPerDay = stats.TotalDuration / int(days)

	id := entry.NewID("yearly-scrobbles", published)
	ee := &entry.Entry{
		ID: id,
		Frontmatter: entry.Frontmatter{
			Title: fmt.Sprintf("%d in Music", published.Year()),
			Description: fmt.Sprintf(
				"In %d, I listened to %d tracks from %d different artists for a total of %.2f hours.",
				published.Year(),
				stats.UniqueTracks,
				stats.UniqueArtists,
				durationFromSeconds(float64(stats.TotalDuration)).Hours(),
			),
			Template: "scrobbles-report",
			Sections: []string{"listens"},
			Properties: map[string]interface{}{
				"category": []string{"yearly-scrobbles"},
			},
			Published: published,
		},
	}

	err = e.SaveEntry(ee)
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, id, "_stats.json")
	err = e.fs.WriteJSON(filename, stats, "update stats")
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}

func (e *Eagle) initScrobbleCron() error {
	_, err := e.cron.AddFunc("CRON_TZ=UTC 00 01 * * *", func() {
		today := time.Now().UTC()
		yesterday := today.AddDate(0, 0, -1)
		year, month, day := yesterday.Date()

		err := e.FetchLastfmScrobbles(year, month, day)
		if err != nil {
			e.Notifier.Error(fmt.Errorf("daily scrobbles cron job: %w", err))
		}

		if today.Day() != 1 {
			// Not the first day of the month, stop.
			return
		}

		err = e.MakeMonthlyScrobblesReport(year, month)
		if err != nil {
			e.Notifier.Error(fmt.Errorf("monthly scrobbles cron job: %w", err))
		}

		if today.Month() != time.January {
			// Not the first month of the year, stop.
			return
		}

		err = e.MakeYearlyScrobblesReport(year)
		if err != nil {
			e.Notifier.Error(fmt.Errorf("yearly scrobbles cron job: %w", err))
		}
	})

	return err
}

type IndividualStats struct {
	Name      string `json:"name"`
	Duration  int    `json:"duration"`
	Scrobbles int    `json:"scrobbles"`
}

type ScrobbleStats struct {
	Start               time.Time          `json:"start"`
	End                 time.Time          `json:"end"`
	TotalDuration       int                `json:"totalDuration"`
	ListeningClock      []int              `json:"listeningClock"`
	Scrobbles           int                `json:"scrobbles"`
	TracksPerDay        int                `json:"tracksPerDay"`
	DurationPerDay      int                `json:"durationPerDay"`
	ScrobblesPerWeekday []int              `json:"scrobblesPerWeekday"`
	UniqueArtists       int                `json:"uniqueArtists"`
	UniqueTracks        int                `json:"uniqueTracks"`
	UniqueAlbums        int                `json:"uniqueAlbums"`
	Artists             []*IndividualStats `json:"artists"`
	Tracks              []*IndividualStats `json:"tracks"`
	Albums              []*IndividualStats `json:"albums"`
}

func statsFromScrobbles(scrobbles []*mf2.FlatHelper) (*ScrobbleStats, error) {
	stats := &ScrobbleStats{
		ListeningClock:      make([]int, 24),
		ScrobblesPerWeekday: make([]int, 7),
		TotalDuration:       0,
	}

	for i := range stats.ListeningClock {
		stats.ListeningClock[i] = 0
	}

	for i := range stats.ScrobblesPerWeekday {
		stats.ScrobblesPerWeekday[i] = 0
	}

	artistsMap := map[string]*IndividualStats{}
	tracksMap := map[string]*IndividualStats{}
	albumsMap := map[string]*IndividualStats{}

	for _, scrobble := range scrobbles {
		artist := scrobble.Sub("author")
		if artist == nil {
			log.S().Warnf("track has no artist: %s", scrobble.Name())
			continue
		}

		t, err := dateparse.ParseStrict(scrobble.String("published"))
		if err != nil {
			return nil, err
		}

		stats.ListeningClock[t.Hour()]++
		stats.ScrobblesPerWeekday[weekdayToIndex(t.Weekday())]++
		stats.Scrobbles++

		duration := scrobble.Int("duration")
		stats.TotalDuration += duration

		name := scrobble.Name()
		artistName := artist.Name()

		key := name + artistName
		if _, ok := tracksMap[key]; !ok {
			tracksMap[key] = &IndividualStats{
				Name:      name,
				Duration:  duration,
				Scrobbles: 0,
			}
		}
		tracksMap[key].Scrobbles++

		if _, ok := artistsMap[artistName]; !ok {
			artistsMap[artistName] = &IndividualStats{
				Name:      artistName,
				Duration:  0,
				Scrobbles: 0,
			}
		}
		artistsMap[artistName].Scrobbles++
		artistsMap[artistName].Duration += duration

		album := scrobble.Sub("album")
		if album != nil {
			albumName := album.Name()
			key := albumName + artistName
			if _, ok := albumsMap[key]; !ok {
				albumsMap[key] = &IndividualStats{
					Name:      albumName,
					Duration:  0,
					Scrobbles: 0,
				}
			}
			albumsMap[key].Scrobbles++
			albumsMap[key].Duration += duration
		}
	}

	stats.Artists, stats.UniqueArtists = sortAndCropStats(artistsMap)
	stats.Tracks, stats.UniqueTracks = sortAndCropStats(tracksMap)
	stats.Albums, stats.UniqueAlbums = sortAndCropStats(albumsMap)

	return stats, nil
}

func sortAndCropStats(m map[string]*IndividualStats) ([]*IndividualStats, int) {
	a := []*IndividualStats{}

	for _, el := range m {
		a = append(a, el)
	}

	sort.SliceStable(a, func(i, j int) bool {
		return a[i].Scrobbles > a[j].Scrobbles
	})

	l := len(a)
	if len(a) > 100 {
		a = a[0:100]
	}
	return a, l
}
