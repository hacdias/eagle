package eagle

import (
	"encoding/xml"
	"time"
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

func (a lastfmArtist) toFlatMF2() map[string]interface{} {
	props := map[string]interface{}{
		"name": a.Name,
	}

	if a.MBID != "" {
		props["mbid"] = a.MBID
	}

	return map[string]interface{}{
		"type":       "h-card",
		"properties": props,
	}
}

type lastfmAlbum struct {
	XMLName xml.Name `xml:"album"`
	MBID    string   `xml:"mbid,attr"`
	Name    string   `xml:",chardata"`
}

func (a lastfmAlbum) toFlatMF2() map[string]interface{} {
	props := map[string]interface{}{
		"name": a.Name,
	}

	if a.MBID != "" {
		props["mbid"] = a.MBID
	}

	return map[string]interface{}{
		"type":       "h-card",
		"properties": props,
	}
}

type lastfmDate struct {
	XMLName xml.Name `xml:"date"`
	UTS     int64    `xml:"uts,attr"`
	Text    string   `xml:",chardata"`
}

func (l lastfmDate) toTime() time.Time {
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
	Duration   time.Duration `xml:"-"`
	Tags       []string      `xml:"-"`
}

func (t *lastfmTrack) DurationOrAverage() time.Duration {
	if t.Duration == 0 {
		return time.Second * 150 // 3.5m
	}

	return t.Duration
}

func (t *lastfmTrack) toFlatMF2() map[string]interface{} {
	props := map[string]interface{}{
		"name":      t.Name,
		"author":    t.Artist.toFlatMF2(),
		"album":     t.Album.toFlatMF2(),
		"published": t.Date.toTime().Format(time.RFC3339),
		"category":  t.Tags,
	}

	if t.URL != "" {
		props["url"] = t.URL
	}

	if t.MBID != "" {
		props["mbid"] = t.MBID
	}

	return map[string]interface{}{
		"type":       "h-cite",
		"properties": props,
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

type lastfmTracksResponse struct {
	XMLName      xml.Name      `xml:"lfm"`
	RecentTracks *lastfmTracks `xml:"recenttracks"`
}

type lastfmTag struct {
	XMLName xml.Name `xml:"tag"`
	Name    string   `xml:"name"`
	URL     string   `xml:"url"`
}

type lastfmTags struct {
	XMLName xml.Name    `xml:"toptags"`
	Tags    []lastfmTag `xml:"tag"`
}

type lastfmTrackInfo struct {
	XMLName  xml.Name   `xml:"track"`
	Name     string     `xml:"name"`
	Duration int64      `xml:"duration"`
	Tags     lastfmTags `xml:"toptags"`
}

type lastfmTrackInfoResponse struct {
	XMLName xml.Name         `xml:"lfm"`
	Track   *lastfmTrackInfo `xml:"track"`
}
