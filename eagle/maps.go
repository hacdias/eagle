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

func (e *Eagle) ProcessLocationMap(ee *entry.Entry) error {
	entryType := ee.Helper().PostType()

	if entryType == mf2.TypeItinerary {
		return e.processItineraryMap(ee)
	}

	if entryType == mf2.TypeCheckin {
		return e.processRegularMap(ee)
	}

	// Do not make a map for others.
	return nil
}

func (e *Eagle) processRegularMap(ee *entry.Entry) error {
	location := ee.Helper().Sub("location")
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

func (e *Eagle) processItineraryMap(ee *entry.Entry) error {
	legs := ee.Helper().Subs("itinerary")
	if legs == nil {
		return errors.New("itinerary has no legs")
	}

	var paths []*geojson.Feature
	var points []*geojson.Feature

	for i, leg := range legs {
		origin := leg.Sub("origin")
		if origin == nil {
			return errors.New("origin is not microformat")
		}

		destination := leg.Sub("destination")
		if destination == nil {
			return errors.New("origin is not microformat")
		}

		ocoord := []float64{
			truncateFloat(origin.Float("longitude")),
			truncateFloat(origin.Float("latitude")),
		}

		// Add the first marker as green for start.
		if i == 0 {
			feature := geojson.NewPointFeature(ocoord)
			feature.SetProperty("marker-color", "#2ecc71")
			points = append(points, feature)
		}

		dcoord := []float64{
			truncateFloat(destination.Float("longitude")),
			truncateFloat(destination.Float("latitude")),
		}

		// Add the marker as blue if it's an intermediate step,
		// or red otherwise.
		feature := geojson.NewPointFeature(dcoord)
		if i == len(legs)-1 {
			feature.SetProperty("marker-color", "#e74c3c")
		} else {
			feature.SetProperty("marker-color", "#3498db")
		}
		points = append(points, feature)

		// Create a line path between the origin and destination.
		// TODO: for flights, make it curvy.
		paths = append(paths, geojson.NewLineStringFeature([][]float64{ocoord, dcoord}))
	}

	fc := geojson.NewFeatureCollection()
	fc.Features = append(fc.Features, paths...)
	fc.Features = append(fc.Features, points...)

	// Get map with GeoJSON and save it
	data, typ, err := e.mapboxGeoJSON(fc)
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, ee.ID, "map."+typ)
	return e.fs.WriteFile(filename, data, "map")
}

func truncateFloat(i float64) float64 {
	return float64(int(i*10000)) / 10000
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
