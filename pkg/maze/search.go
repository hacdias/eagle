package maze

import (
	"errors"
	"io"
	"net/url"

	geojson "github.com/paulmach/go.geojson"
)

func (l *Maze) photonSearch(lang, query string) (*Location, error) {
	uv := url.Values{}
	uv.Set("q", query)
	uv.Set("lang", lang)

	res, err := l.httpClient.Get("https://photon.komoot.io/api/?" + uv.Encode())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	fc, err := geojson.UnmarshalFeatureCollection(data)
	if err != nil {
		return nil, err
	}

	if len(fc.Features) < 1 {
		return nil, errors.New("features missing from request")
	}

	f := fc.Features[0]
	city := f.PropertyMustString("city", "")
	state := f.PropertyMustString("state", f.PropertyMustString("county", ""))
	country := f.PropertyMustString("country", "")

	if city == "" && state == "" && country == "" {
		return nil, errors.New("no useful information found")
	}

	loc := &Location{
		Name:     query,
		Locality: city,
		Region:   state,
		Country:  country,
	}

	if f.Geometry != nil && len(f.Geometry.Point) == 2 {
		loc.Longitude = f.Geometry.Point[0]
		loc.Latitude = f.Geometry.Point[1]
	}

	return loc, nil
}
