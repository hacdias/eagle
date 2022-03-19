package eagle

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/entry"
)

type Lastfm struct {
	*config.Lastfm
}

func (l *Lastfm) Fetch(year int, month time.Month, day int) (entry.Tracks, error) {
	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	from := midnight.Unix()
	to := midnight.Unix() + 86400 // +1 Day

	tracks := []*entry.Track{}
	page := 1

	for {
		res, err := l.recentTracks(page, from, to)
		if err != nil {
			return nil, err
		}

		for _, track := range res.convert() {
			if track.Duration == 0 {
				info, err := l.trackInfo(track)
				if err == nil {
					track.Duration = time.Duration(info.Duration) * time.Millisecond
				} // When this fails, we assume an average time of 3m30s.
			}

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

func (l *Lastfm) trackInfo(t *entry.Track) (*lastfmTrackInfo, error) {
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

	tracks, err := e.lastfm.Fetch(year, month, day)
	if err != nil {
		return err
	}

	if len(tracks) == 0 {
		return nil
	}

	id := entry.NewID("daily-scrobbles", time.Date(year, month, day, 0, 0, 0, 0, time.UTC))

	ee := &entry.Entry{
		ID: id,
		Frontmatter: entry.Frontmatter{
			Title:              "Daily Scrobbles",
			Template:           "daily-scrobbles",
			Sections:           []string{"listens"},
			NoSendInteractions: true,
			Properties: map[string]interface{}{
				"category": []string{"daily-scrobbles"},
			},
			Published: time.Date(year, month, day, 23, 59, 59, 0, time.UTC),
		},
	}

	err = e.SaveEntry(ee)
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, id, "_scrobbles.json")
	err = e.fs.WriteJSON(filename, tracks, "update scrobbles")
	if err != nil {
		return err
	}

	e.RemoveCache(ee)
	return nil
}
