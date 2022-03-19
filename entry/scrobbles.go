package entry

import (
	"encoding/json"
	"sort"
	"time"
)

type Artist struct {
	MBID string `json:"mbid"`
	Name string `json:"name"`
}

type ArtistWithDuration struct {
	Artist
	Duration time.Duration
}

type Album struct {
	MBID string `json:"mbid"`
	Name string `json:"name"`
}

type Track struct {
	Name     string        `json:"name"`
	MBID     string        `json:"mbid"`
	URL      string        `json:"url"`
	Artist   Artist        `json:"artist"`
	Album    Album         `json:"album"`
	Date     time.Time     `json:"date"`
	Duration time.Duration `json:"duration"`
	Image    string        `json:"image"`
}

func (t Track) DurationOrAverage() time.Duration {
	if t.Duration == 0 {
		return time.Second * 150 // 3.5m
	}

	return t.Duration
}

type TrackWithPlays struct {
	Track
	Plays []time.Time
}

func (ts *TrackWithPlays) TotalDuration() time.Duration {
	return ts.DurationOrAverage() * time.Duration(len(ts.Plays))
}

type Artists []*ArtistWithDuration

type Tracks []*Track

func (ts Tracks) TotalDuration() time.Duration {
	var dur time.Duration
	for _, track := range ts {
		dur += track.DurationOrAverage()
	}
	return dur
}

func (s Tracks) ByTrack() []*TrackWithPlays {
	tracksByKey := map[string]*TrackWithPlays{}

	for _, track := range s {
		key := track.Name + track.Artist.Name
		if _, ok := tracksByKey[key]; !ok {
			tracksByKey[key] = &TrackWithPlays{
				Track: *track,
			}
		}

		tracksByKey[key].Plays = append(tracksByKey[key].Plays, track.Date)
	}

	tracks := []*TrackWithPlays{}
	for _, track := range tracksByKey {
		tracks = append(tracks, track)
	}

	sort.Slice(tracks, func(i, j int) bool {
		return tracks[i].Name < tracks[j].Name
	})

	return tracks
}

func (ts Tracks) ByArtist() Artists {
	artistsByName := map[string]*ArtistWithDuration{}

	for _, track := range ts {
		key := track.Artist.Name
		if _, ok := artistsByName[key]; !ok {
			artistsByName[key] = &ArtistWithDuration{
				Artist: track.Artist,
			}
		}

		artistsByName[key].Duration += track.DurationOrAverage()
	}

	artists := []*ArtistWithDuration{}
	for _, artist := range artistsByName {
		artists = append(artists, artist)
	}

	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Duration > artists[j].Duration
	})

	return artists
}

func (ts Tracks) ListeningClock() string {
	hours := make([]float64, 24)

	for _, track := range ts {
		h := track.Date.Hour()
		hours[h] += track.DurationOrAverage().Minutes()
	}

	v, _ := json.Marshal(hours)
	return string(v)
}
