package lastfm

import (
	"encoding/xml"
	"strings"
	"time"
)

type NameAndID struct {
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

type Artist = NameAndID
type Album = NameAndID

type Track struct {
	NameAndID
	URL           string    `json:"url,omitempty"`
	Artist        Artist    `json:"artist"`
	Album         Album     `json:"album"`
	OriginalImage string    `json:"originalImage,omitempty"`
	Image         string    `json:"image,omitempty"`
	Date          time.Time `json:"date"`
}

type image struct {
	XMLName xml.Name `xml:"image"`
	Size    string   `xml:"size,attr"`
	URL     string   `xml:",chardata"`
}

type artist struct {
	XMLName xml.Name `xml:"artist"`
	MBID    string   `xml:"mbid,attr"`
	Name    string   `xml:",chardata"`
}

func (a artist) convert() Artist {
	return Artist{
		Name: a.Name,
		MBID: a.MBID,
	}
}

type album struct {
	XMLName xml.Name `xml:"album"`
	MBID    string   `xml:"mbid,attr"`
	Name    string   `xml:",chardata"`
}

func (a album) convert() Album {
	return Album{
		Name: a.Name,
		MBID: a.MBID,
	}
}

type date struct {
	XMLName xml.Name `xml:"date"`
	UTS     int64    `xml:"uts,attr"`
	Text    string   `xml:",chardata"`
}

func (l date) convert() time.Time {
	return time.Unix(l.UTS, 0)
}

type Tag struct {
	XMLName xml.Name `xml:"tag"`
	Name    string   `xml:"name"`
	URL     string   `xml:"url"`
}

type track struct {
	XMLName    xml.Name `xml:"track"`
	NowPlaying bool     `xml:"nowplaying,attr"`
	Name       string   `xml:"name"`
	MBID       string   `xml:"mbid"`
	URL        string   `xml:"url"`
	Artist     artist   `xml:"artist"`
	Album      album    `xml:"album"`
	Images     []image  `xml:"image"`
	Date       date     `xml:"date"`
}

func (t *track) convert() *Track {
	track := &Track{
		NameAndID: NameAndID{
			Name: t.Name,
			MBID: t.MBID,
		},
		URL:    t.URL,
		Artist: t.Artist.convert(),
		Album:  t.Album.convert(),
		Date:   t.Date.convert(),
	}

	images := map[string]string{}
	for _, i := range t.Images {
		if i.Size != "" && i.URL != "" {
			images[i.Size] = i.URL
		}
	}

	for _, v := range []string{"extralarge", "large", "medium", "small"} {
		if img, ok := images[v]; ok {
			track.OriginalImage = img
			break
		}
	}

	return track
}

type tracks struct {
	XMLName    xml.Name `xml:"recenttracks"`
	Tracks     []*track `xml:"track"`
	Page       int64    `xml:"page,attr"`
	PerPage    int64    `xml:"perPage,attr"`
	TotalPages int64    `xml:"totalPages,attr"`
	Total      int64    `xml:"total,attr"`
}

type recentTracksResponse struct {
	XMLName      xml.Name `xml:"lfm"`
	RecentTracks *tracks  `xml:"recenttracks"`
}

type tags struct {
	XMLName xml.Name `xml:"toptags"`
	Tags    []Tag    `xml:"tag"`
}

func (t tags) convert() []string {
	tags := []string{}
	for _, v := range t.Tags {
		tags = append(tags, strings.ToLower(v.Name))
	}
	return tags
}

type trackInfo struct {
	XMLName  xml.Name `xml:"track"`
	Name     string   `xml:"name"`
	Duration int64    `xml:"duration"`
	Tags     tags     `xml:"toptags"`
}

type trackInfoResponse struct {
	XMLName xml.Name   `xml:"lfm"`
	Track   *trackInfo `xml:"track"`
}
