package maze

import (
	"errors"
	"fmt"
	"io"
	"net/url"

	geojson "github.com/paulmach/go.geojson"
)

func (l *Maze) photonReverse(lang string, lon, lat float64) (*Location, error) {
	uv := url.Values{}
	uv.Set("lat", fmt.Sprintf("%v", lat))
	uv.Set("lon", fmt.Sprintf("%v", lon))
	uv.Set("lang", lang)

	res, err := l.httpClient.Get("https://photon.komoot.io/reverse?" + uv.Encode())
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

	return &Location{
		Latitude:  lat,
		Longitude: lon,
		Locality:  city,
		Region:    state,
		Country:   country,
	}, nil
}
