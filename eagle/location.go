package eagle

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	gogeouri "git.jlel.se/jlelse/go-geouri"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/entry/mf2"
	geojson "github.com/paulmach/go.geojson"
)

func (e *Eagle) ProcessLocation(ee *entry.Entry) error {
	if ee.Properties == nil {
		return nil
	}

	mm := ee.Helper()

	locationStr := mm.String("location")
	if locationStr == "" {
		return nil
	}

	var (
		location map[string]interface{}
		err      error
	)

	if strings.HasPrefix(locationStr, "geo:") {
		var geo *gogeouri.Geo
		geo, err = gogeouri.Parse(locationStr)
		if err != nil {
			return err
		}

		location, err = e.photonReverse(geo.Longitude, geo.Latitude)
	} else if strings.HasPrefix(locationStr, "airport:") {
		// https://www.aviowiki.com/docs/airports/free-api-endpoints/airport-search/
	} else if strings.HasPrefix(locationStr, "name:") {
		locationStr = strings.TrimPrefix(locationStr, "name:")
		location, err = e.photonSearch(locationStr)
	}

	// TODO: Maybe detect it is itinerary and replace origin and destination by a h-adr with name property.
	// Add properties[location] = properties[itinerary][destination]

	if err != nil {
		return err
	}

	if location == nil {
		return nil
	}

	_, err = e.TransformEntry(ee.ID, func(ee *entry.Entry) (*entry.Entry, error) {
		ee.Properties["location"] = location
		return ee, nil
	})

	return err
}

func (e *Eagle) photonReverse(lon, lat float64) (map[string]interface{}, error) {
	uv := url.Values{}
	uv.Set("lat", fmt.Sprintf("%v", lat))
	uv.Set("lon", fmt.Sprintf("%v", lon))
	uv.Set("lang", e.Config.Site.Language)

	res, err := e.httpClient.Get("https://photon.komoot.io/reverse?" + uv.Encode())
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
		"latitude":  lat,
		"longitude": lon,
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

func (e *Eagle) photonSearch(query string) (map[string]interface{}, error) {
	uv := url.Values{}
	uv.Set("q", query)
	uv.Set("lang", e.Config.Site.Language)

	res, err := e.httpClient.Get("https://photon.komoot.io/api/?" + uv.Encode())
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

	props := map[string]interface{}{}

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

func (e *Eagle) ProcessLocationMap(ee *entry.Entry) error {
	mm := ee.Helper()

	if mm.PostType() != mf2.TypeCheckin {
		// Only get location maps for checkins.
		return nil
	}

	location := mm.Sub("location")
	if location == nil {
		return nil
	}

	latitude := location.Properties.Float("latitude")
	longitude := location.Properties.Float("longitude")

	data, typ, err := e.mapboxStatic(longitude, latitude)
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, ee.ID, "map."+typ)
	return e.fs.WriteFile(filename, data, "map")
}

func (e *Eagle) mapboxStatic(lon, lat float64) ([]byte, string, error) {
	if e.Config.MapBox == nil {
		return nil, "", errors.New("mapbox details not provided")
	}

	path := fmt.Sprintf(
		"https://api.mapbox.com/styles/v1/mapbox/%s/static/pin-l+%s(%f,%f)/%f,%f,%d,0/%s",
		e.Config.MapBox.MapStyle,
		e.Config.MapBox.PinColor,
		lon,
		lat,
		lon,
		lat,
		e.Config.MapBox.Zoom,
		e.Config.MapBox.Size,
	)

	if e.Config.MapBox.Use2X {
		path += "@2x"
	}

	path += "?access_token=" + e.Config.MapBox.AccessToken

	res, err := e.httpClient.Get(path)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	typ := "png"
	if strings.Contains(res.Header.Get("Content-Type"), "jpeg") {
		typ = "jpeg"
	}

	return data, typ, nil
}
