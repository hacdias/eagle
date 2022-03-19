package eagle

import (
	"encoding/xml"
	"time"

	"github.com/hacdias/eagle/v3/entry"
)

type lastfmImage struct {
	XMLName xml.Name `xml:"image"`
	Size    string   `xml:"size,attr"`
	URL     string   `xml:",chardata"`
}

type lastfmArtist struct {
	XMLName xml.Name `xml:"artist"`
	MBID    string   `xml:"mbid,attr"`
	Name    string   `xml:",chardata"`
}

func (a lastfmArtist) convert() entry.Artist {
	return entry.Artist{
		Name: a.Name,
		MBID: a.MBID,
	}
}

type lastfmAlbum struct {
	XMLName xml.Name `xml:"album"`
	MBID    string   `xml:"mbid,attr"`
	Name    string   `xml:",chardata"`
}

func (a lastfmAlbum) convert() entry.Album {
	return entry.Album{
		Name: a.Name,
		MBID: a.MBID,
	}
}

type lastfmDate struct {
	XMLName xml.Name `xml:"date"`
	UTS     int64    `xml:"uts,attr"`
	Text    string   `xml:",chardata"`
}

func (l lastfmDate) convert() time.Time {
	return time.Unix(l.UTS, 0)
}

type lastfmTrack struct {
	XMLName    xml.Name      `xml:"track"`
	NowPlaying bool          `xml:"nowplaying,attr"`
	Name       string        `xml:"name"`
	MBID       string        `xml:"mbid"`
	URL        string        `xml:"url"`
	Artist     lastfmArtist  `xml:"artist"`
	Album      lastfmAlbum   `xml:"album"`
	Images     []lastfmImage `xml:"image"`
	Date       lastfmDate    `xml:"date"`
}

func (t *lastfmTrack) convert() *entry.Track {
	return &entry.Track{
		Name:     t.Name,
		MBID:     t.MBID,
		URL:      t.URL,
		Artist:   t.Artist.convert(),
		Album:    t.Album.convert(),
		Date:     t.Date.convert(),
		Duration: 0,
		Image:    "",
	}
}

type lastfmTracks struct {
	XMLName    xml.Name       `xml:"recenttracks"`
	Tracks     []*lastfmTrack `xml:"track"`
	Page       int64          `xml:"page,attr"`
	PerPage    int64          `xml:"perPage,attr"`
	TotalPages int64          `xml:"totalPages,attr"`
	Total      int64          `xml:"total,attr"`
}

func (ts *lastfmTracks) convert() []*entry.Track {
	tracks := []*entry.Track{}

	for _, track := range ts.Tracks {
		if track.NowPlaying {
			continue
		}

		tracks = append(tracks, track.convert())
	}

	return tracks
}

type lastfmTracksResponse struct {
	XMLName      xml.Name      `xml:"lfm"`
	RecentTracks *lastfmTracks `xml:"recenttracks"`
}

type lastfmTrackInfo struct {
	XMLName  xml.Name `xml:"track"`
	Name     string   `xml:"name"`
	Duration int64    `xml:"duration"`
}

type lastfmTrackInfoResponse struct {
	XMLName xml.Name         `xml:"lfm"`
	Track   *lastfmTrackInfo `xml:"track"`
}
