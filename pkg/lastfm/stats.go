package lastfm

import (
	"sort"
	"time"
)

const AverageDuration = time.Second * 150

type IndividualStats struct {
	Track     Track
	Name      string
	Scrobbles int
}

type ArtistsStats = IndividualStats

type TracksStats = IndividualStats

type AlbumsStats = IndividualStats

type Stats struct {
	TotalDuration  time.Duration
	TotalScrobbles int
	TotalTracks    int
	TotalArtists   int
	TotalAlbums    int
	ListeningClock []int
	WeekdayClock   []int
	Artists        []*IndividualStats
	Tracks         []*IndividualStats
	Albums         []*IndividualStats
}

func ScrobblesStatistics(scrobbles []*Track) *Stats {
	stats := &Stats{
		ListeningClock: make([]int, 24),
		WeekdayClock:   make([]int, 7),
		TotalScrobbles: len(scrobbles),
		TotalDuration:  AverageDuration * time.Duration(len(scrobbles)),
	}

	for i := range stats.ListeningClock {
		stats.ListeningClock[i] = 0
	}

	for i := range stats.WeekdayClock {
		stats.WeekdayClock[i] = 0
	}

	artistsMap := map[string]*IndividualStats{}
	tracksMap := map[string]*IndividualStats{}
	albumsMap := map[string]*IndividualStats{}

	for _, s := range scrobbles {
		stats.ListeningClock[s.Date.Hour()]++
		stats.WeekdayClock[s.Date.Weekday()]++

		key := s.Name + s.Artist.Name
		if _, ok := tracksMap[key]; !ok {
			tracksMap[key] = &IndividualStats{
				Name:      s.Name,
				Scrobbles: 0,
				Track:     *s,
			}
			stats.Tracks = append(stats.Tracks, tracksMap[key])
		}
		tracksMap[key].Scrobbles++

		if _, ok := artistsMap[s.Artist.Name]; !ok {
			artistsMap[s.Artist.Name] = &IndividualStats{
				Name:      s.Artist.Name,
				Scrobbles: 0,
				Track:     *s,
			}
			stats.Artists = append(stats.Artists, artistsMap[s.Artist.Name])
		}
		artistsMap[s.Artist.Name].Scrobbles++

		if s.Album.Name != "" {
			key := s.Album.Name + s.Artist.Name
			if _, ok := albumsMap[key]; !ok {
				albumsMap[key] = &IndividualStats{
					Name:      s.Album.Name,
					Scrobbles: 0,
					Track:     *s,
				}
				stats.Albums = append(stats.Albums, albumsMap[key])
			}
			albumsMap[key].Scrobbles++
		}
	}

	stats.TotalTracks = len(stats.Tracks)
	stats.TotalArtists = len(stats.Artists)
	stats.TotalAlbums = len(stats.Albums)

	sort.SliceStable(stats.Tracks, sortIndividualStats(stats.Tracks))
	sort.SliceStable(stats.Artists, sortIndividualStats(stats.Artists))
	sort.SliceStable(stats.Albums, sortIndividualStats(stats.Albums))

	return stats
}

func sortIndividualStats(m []*IndividualStats) func(i, j int) bool {
	return func(i, j int) bool {
		return m[i].Scrobbles > m[j].Scrobbles
	}
}
