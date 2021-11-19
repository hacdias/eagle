package eagle

import (
	"errors"
	"path/filepath"

	"github.com/hacdias/eagle/v2/entry"
	"github.com/karlseguin/typed"
	geojson "github.com/paulmach/go.geojson"
)

// TODO: DECOUPLE PROCESS ITINERARY FROM GENERATING THE GEOJSON MAP.
func (e *Eagle) processItinerary(ee *entry.Entry) error {
	mm := ee.Helper()

	var legs []typed.Typed
	if leg, ok := mm.Properties.ObjectIf(mm.TypeProperty()); ok {
		legs = append(legs, leg)
	} else if llegs, ok := mm.Properties.ObjectsIf(mm.TypeProperty()); ok {
		legs = llegs
	} else {
		return errors.New("itinerary without legs")
	}

	if len(legs) == 0 {
		return errors.New("itinerary without legs")
	}

	var lastDest map[string]interface{}
	fc := geojson.NewFeatureCollection()

	for i, leg := range legs {
		props, ok := leg.ObjectIf("properties")
		if !ok {
			return errors.New("leg missing properties")
		}

		transitType := props.String("transit-type")

		if _, ok := props.ObjectIf("origin"); ok {
			// This entry was most likely already processed.
			// Otherwise, origin wouldn't be a map.
			return nil
		}

		// Origin
		originCoord, _, err := e.parseLocationCoord(props, "origin", transitType)
		if err != nil {
			return err
		}

		if i == 0 {
			feature := geojson.NewPointFeature(originCoord)
			feature.SetProperty("marker-color", "#2ecc71")
			fc.Features = append(fc.Features, feature)
		}

		// Destination
		destCoord, loc, err := e.parseLocationCoord(props, "destination", transitType)
		if err != nil {
			return err
		}
		lastDest = loc

		feature := geojson.NewPointFeature(destCoord)
		if i == len(legs)-1 {
			feature.SetProperty("marker-color", "#e74c3c")
		} else {
			feature.SetProperty("marker-color", "#3498db")
		}
		fc.Features = append(fc.Features, feature)
		fc.Features = append(fc.Features, geojson.NewLineStringFeature([][]float64{originCoord, destCoord}))
	}

	_, err := e.TransformEntry(ee.ID, func(ee *entry.Entry) (*entry.Entry, error) {
		if lastDest != nil {
			ee.Properties["location"] = lastDest
		}

		if len(legs) == 1 {
			ee.Properties[mm.TypeProperty()] = legs[0]
		} else {
			ee.Properties[mm.TypeProperty()] = legs
		}
		return ee, nil
	})
	if err != nil {
		return err
	}

	// Get map with GeoJSON and save it
	data, typ, err := e.mapboxGeoJSON(fc)
	if err != nil {
		return err
	}

	filename := filepath.Join(ContentDirectory, ee.ID, "map."+typ)
	return e.fs.WriteFile(filename, data, "map")
}

func (e *Eagle) parseLocationCoord(props typed.Typed, prop, transitType string) ([]float64, map[string]interface{}, error) {
	origin := props.String(prop)
	if origin == "" {
		return nil, nil, errors.New("origin missing")
	}

	loc, err := e.parseLocation(origin, transitType == "air")
	if err != nil {
		return nil, nil, err
	}
	props[prop] = loc

	locProps, ok := loc["properties"].(map[string]interface{})
	if !ok {
		return nil, nil, errors.New("location properties must be map")
	}

	lon, ok := locProps["longitude"].(float64)
	if !ok {
		return nil, nil, errors.New("longitude is invalid")
	}

	lat, ok := locProps["latitude"].(float64)
	if !ok {
		return nil, nil, errors.New("latitude is invalid")
	}

	return []float64{truncateFloat(lon), truncateFloat(lat)}, loc, nil
}

func truncateFloat(i float64) float64 {
	return float64(int(i*10000)) / 10000
}
