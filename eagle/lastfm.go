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
			Description: fmt.Sprintf("Listened to %d tracks %.2f hours", len(tracks), totalDuration.Hours()),
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

func getMondayOfWeek(year int, month time.Month, day int) time.Time {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, -1)
	}

	return t
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

func (e *Eagle) MakeWeeklyScrobblesReport(year int, month time.Month, day int) error {
	start := getMondayOfWeek(year, month, day)
	end := start.AddDate(0, 0, 7)

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

	_, week := published.ISOWeek()

	id := entry.NewID("weekly-scrobbles", published)
	ee := &entry.Entry{
		ID: id,
		Frontmatter: entry.Frontmatter{
			Title:    fmt.Sprintf("Week %d in Music", week),
			Template: "scrobbles-report",
			Sections: []string{"listens"},
			Properties: map[string]interface{}{
				"category": []string{"weekly-scrobbles"},
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

type IndividualStats struct {
	Name      string `json:"name"`
	Duration  int    `json:"duration"`
	Scrobbles int    `json:"scrobbles"`
}

type ScrobbleStats struct {
	Start               time.Time          `json:"start"`
	End                 time.Time          `json:"end"`
	ListeningClock      []int              `json:"listeningClock"`
	ScrobblesPerWeekday []int              `json:"scrobblesPerWeekday"`
	Artists             []*IndividualStats `json:"artists"`
	Tracks              []*IndividualStats `json:"tracks"`
	Albums              []*IndividualStats `json:"albums"`
}

func statsFromScrobbles(scrobbles []*mf2.FlatHelper) (*ScrobbleStats, error) {
	stats := &ScrobbleStats{
		ListeningClock:      make([]int, 24),
		ScrobblesPerWeekday: make([]int, 7),
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

		duration := scrobble.Int("duration")

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

	for _, artist := range artistsMap {
		stats.Artists = append(stats.Artists, artist)
	}

	sort.SliceStable(stats.Artists, func(i, j int) bool {
		return stats.Artists[i].Scrobbles > stats.Artists[j].Scrobbles
	})

	for _, track := range tracksMap {
		stats.Tracks = append(stats.Tracks, track)
	}

	sort.SliceStable(stats.Tracks, func(i, j int) bool {
		return stats.Tracks[i].Scrobbles > stats.Tracks[j].Scrobbles
	})

	for _, album := range albumsMap {
		stats.Albums = append(stats.Albums, album)
	}

	sort.SliceStable(stats.Albums, func(i, j int) bool {
		return stats.Albums[i].Scrobbles > stats.Albums[j].Scrobbles
	})

	return stats, nil
}
