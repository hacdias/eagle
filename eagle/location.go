package eagle

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

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

	location, err := e.parseLocation(locationStr, false)
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

func (e *Eagle) parseLocation(str string, isAirport bool) (map[string]interface{}, error) {
	if strings.HasPrefix(str, "geo:") {
		return e.loctools.FromGeoURI(e.Config.Site.Language, str)
	}

	if isAirport {
		return e.loctools.Airport(str)
	}

	return e.loctools.Search(e.Config.Site.Language, str)
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

func (e *Eagle) mapboxGeoJSON(geojson *geojson.FeatureCollection) ([]byte, string, error) {
	if e.Config.MapBox == nil {
		return nil, "", errors.New("mapbox details not provided")
	}

	raw, err := geojson.MarshalJSON()
	if err != nil {
		return nil, "", err
	}

	path := fmt.Sprintf(
		"https://api.mapbox.com/styles/v1/mapbox/%s/static/geojson(%s)/auto/%s",
		e.Config.MapBox.MapStyle,
		url.QueryEscape(string(raw)),
		e.Config.MapBox.Size,
	)

	if e.Config.MapBox.Use2X {
		path += "@2x"
	}

	path += "?padding=20&access_token=" + e.Config.MapBox.AccessToken

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
