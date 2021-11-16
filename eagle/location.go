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

func (e *Eagle) processLocation(ee *entry.Entry) error {
	if ee.Properties == nil {
		return nil
	}

	mm := ee.Helper()

	geouri := mm.String("location")
	if geouri == "" {
		return nil
	}

	geo, err := gogeouri.Parse(geouri)
	if err != nil {
		return err
	}

	location, err := e.photonReverse(geo.Longitude, geo.Latitude)
	if err != nil {
		return err
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
	state := f.PropertyMustString("state", "")
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
