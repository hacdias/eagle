package lastfm

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hacdias/eagle/v4/log"
)

type LastFm struct {
	key  string
	user string
}

func NewLastFm(key, user string) *LastFm {
	return &LastFm{
		key:  key,
		user: user,
	}
}

func (l *LastFm) Fetch(year int, month time.Month, day int) ([]*Track, error) {
	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	from := midnight.Unix()
	to := midnight.Unix() + 86400 // +1 Day

	tracks := []*Track{}
	page := 1

	for {
		res, err := l.recentTracks(page, from, to)
		if err != nil {
			return nil, err
		}

		for _, rawTrack := range res.Tracks {
			if rawTrack.NowPlaying {
				continue
			}

			track := rawTrack.convert()

			info, err := l.trackInfo(rawTrack)
			if err == nil {
				track.Duration = time.Duration(info.Duration) * time.Millisecond
				track.Tags = info.Tags.convert()
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

func (l *LastFm) recentTracks(page int, from, to int64) (*tracks, error) {
	limit := 200
	u, err := url.Parse("https://ws.audioscrobbler.com/2.0/")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("method", "user.getrecenttracks")
	q.Set("user", l.user)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page))
	q.Set("api_key", l.key)

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

	var response *recentTracksResponse
	err = xml.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.RecentTracks, nil
}

func (l *LastFm) trackInfo(t *track) (*trackInfo, error) {
	u, err := url.Parse("https://ws.audioscrobbler.com/2.0/")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("method", "track.getInfo")
	q.Set("api_key", l.key)

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

	var response *trackInfoResponse
	err = xml.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	if response.Track == nil {
		return nil, fmt.Errorf("response is nil")
	}

	return response.Track, nil
}
