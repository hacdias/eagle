package eagle

import (
	"errors"
	"strings"

	"github.com/hacdias/eagle/v2/entry"
	"github.com/hacdias/eagle/v2/entry/mf2"
	"github.com/hacdias/eagle/v2/loctools"
	"github.com/karlseguin/typed"
)

func (e *Eagle) ProcessLocation(ee *entry.Entry) error {
	if ee.Properties == nil {
		return nil
	}

	if ee.Helper().PostType() == mf2.TypeItinerary {
		return e.processItineraryLocations(ee)
	}

	locationStr, ok := ee.Properties["location"].(string)
	if locationStr == "" || !ok {
		return nil
	}

	location, err := e.parseLocation(locationStr)
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

func (e *Eagle) processItineraryLocations(ee *entry.Entry) error {
	if ee.Properties == nil {
		return nil
	}

	props := typed.Typed(ee.Properties)

	var legs []typed.Typed

	if v, ok := props.ObjectIf("itinerary"); ok {
		legs = []typed.Typed{v}
	} else if vv, ok := props.ObjectsIf("itinerary"); ok {
		legs = vv
	} else {
		return errors.New("itinerary has no legs")
	}

	if len(legs) == 0 {
		return errors.New("itinerary has no legs")
	}

	var lastDest map[string]interface{}

	for _, leg := range legs {
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

		_, err := e.parseItineraryLocation(props, "origin", transitType)
		if err != nil {
			return err
		}

		loc, err := e.parseItineraryLocation(props, "destination", transitType)
		if err != nil {
			return err
		}
		lastDest = loc
	}

	_, err := e.TransformEntry(ee.ID, func(ee *entry.Entry) (*entry.Entry, error) {
		if lastDest != nil {
			ee.Properties["location"] = lastDest
		}

		if len(legs) == 1 {
			ee.Properties["itinerary"] = legs[0]
		} else {
			ee.Properties["itinerary"] = legs
		}

		return ee, nil
	})

	return err
}

func (e *Eagle) parseItineraryLocation(props typed.Typed, prop, transitType string) (map[string]interface{}, error) {
	str := props.String(prop)
	if str == "" {
		return nil, errors.New(prop + " missing")
	}

	var (
		location map[string]interface{}
		err      error
	)

	if transitType == "air" {
		location, err = e.parseAirportLocation(str)
	} else {
		location, err = e.parseLocation(str)
	}

	if err != nil {
		return nil, err
	}
	props[prop] = location
	return location, nil
}

func (e *Eagle) parseAirportLocation(str string) (map[string]interface{}, error) {
	var code string

	if strings.Contains(str, "(") {
		str = strings.TrimSpace(str)
		strs := strings.Split(str, "(")
		code = strs[len(strs)-1]
		code = strings.Replace(code, ")", "", 1)
	} else {
		code = str
	}

	loc, err := e.loctools.Airport(code)
	if err != nil {
		return nil, err
	}

	loc.Name = str
	location, err := loc, nil
	if err != nil {
		return nil, err
	}

	return locationToMultiformat(location), nil
}

func (e *Eagle) parseLocation(str string) (map[string]interface{}, error) {
	var (
		location *loctools.Location
		err      error
	)

	if strings.HasPrefix(str, "geo:") {
		location, err = e.loctools.FromGeoURI(e.Config.Site.Language, str)
	} else {
		location, err = e.loctools.Search(e.Config.Site.Language, str)
	}

	if err != nil {
		return nil, err
	}

	return locationToMultiformat(location), nil
}

func locationToMultiformat(loc *loctools.Location) map[string]interface{} {
	props := map[string]interface{}{
		"latitude":  loc.Latitude,
		"longitude": loc.Longitude,
	}

	if loc.Name != "" {
		props["name"] = loc.Name
	}

	if loc.Locality != "" {
		props["locality"] = loc.Locality
	}

	if loc.Region != "" {
		props["region"] = loc.Region
	}

	if loc.Country != "" {
		props["country-name"] = loc.Country
	}

	return map[string]interface{}{
		"type":       "h-adr",
		"properties": props,
	}
}
