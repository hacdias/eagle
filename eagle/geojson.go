package eagle

import (
	"errors"
	"path/filepath"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/entry/mf2"
	geojson "github.com/paulmach/go.geojson"
)

func (e *Eagle) GenerateGeoJSON(ee *entry.Entry) error {
	entryType := ee.Helper().PostType()

	if entryType == mf2.TypeItinerary {
		return e.generateItineraryGeoJSON(ee)
	}

	return nil
}

func (e *Eagle) generateItineraryGeoJSON(ee *entry.Entry) error {
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

	raw, err := fc.MarshalJSON()
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, ee.ID, "_geo.json")
	return e.fs.WriteFile(filename, raw, "geo json")
}

func truncateFloat(i float64) float64 {
	return float64(int(i*10000)) / 10000
}
