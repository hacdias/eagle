package loctools

import (
	"errors"
	"io"
	"net/url"

	geojson "github.com/paulmach/go.geojson"
)

func (l *LocTools) photonSearch(lang, query string) (map[string]interface{}, error) {
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

	props := map[string]interface{}{
		"name": query,
	}

	if f.Geometry != nil && len(f.Geometry.Point) == 2 {
		props["longitude"] = f.Geometry.Point[0]
		props["latitude"] = f.Geometry.Point[1]
	}

	if city != "" {
		props["locality"] = city
	}

	if state != "" {
		props["region"] = state
	}

	if country != "" {
		props["country-name"] = country
	}

	return map[string]interface{}{
		"properties": props,
		"type":       "h-adr",
	}, nil
}
