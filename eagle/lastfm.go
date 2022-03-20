package eagle

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/entry"
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

	id := entry.NewID("daily-scrobbles", time.Date(year, month, day, 0, 0, 0, 0, time.UTC))

	ee := &entry.Entry{
		ID: id,
		Frontmatter: entry.Frontmatter{
			Sections: []string{"listens"},
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
