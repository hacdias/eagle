package hooks

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/pkg/maze"
	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/karlseguin/typed"
)

type LocationFetcher struct {
	language string
	fs       *fs.FS
	maze     *maze.Maze
}

func NewLocationFetcher(fs *fs.FS, language string) *LocationFetcher {
	return &LocationFetcher{
		language: language,
		fs:       fs,
		maze: maze.NewMaze(&http.Client{
			Timeout: time.Minute,
		}),
	}
}

func (l *LocationFetcher) EntryHook(e *eagle.Entry, isNew bool) error {
	if e.Listing != nil {
		return nil
	}

	return l.FetchLocation(e)
}

func (l *LocationFetcher) FetchLocation(e *eagle.Entry) error {
	if e.Properties == nil {
		return nil
	}

	if e.Helper().PostType() == mf2.TypeItinerary {
		return l.processItineraryLocations(e)
	}

	locationStr, ok := e.Properties["location"].(string)
	if locationStr == "" || !ok {
		return nil
	}

	location, err := l.parseLocation(locationStr)
	if err != nil {
		return err
	}

	if location == nil {
		return nil
	}

	_, err = l.fs.TransformEntry(e.ID, func(ee *eagle.Entry) (*eagle.Entry, error) {
		ee.Properties["location"] = location
		return ee, nil
	})

	return err
}

func (l *LocationFetcher) processItineraryLocations(e *eagle.Entry) error {
	if e.Properties == nil {
		return nil
	}

	props := typed.Typed(e.Properties)

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

		_, err := l.parseItineraryLocation(props, "origin", transitType)
		if err != nil {
			return err
		}

		loc, err := l.parseItineraryLocation(props, "destination", transitType)
		if err != nil {
			return err
		}
		lastDest = loc
	}

	_, err := l.fs.TransformEntry(e.ID, func(ee *eagle.Entry) (*eagle.Entry, error) {
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

func (l *LocationFetcher) parseItineraryLocation(props typed.Typed, prop, transitType string) (map[string]interface{}, error) {
	str := props.String(prop)
	if str == "" {
		return nil, errors.New(prop + " missing")
	}

	var (
		location map[string]interface{}
		err      error
	)

	if transitType == "air" {
		location, err = l.parseAirportLocation(str)
	} else {
		location, err = l.parseLocation(str)
	}

	if err != nil {
		return nil, err
	}

	props[prop] = location
	return location, nil
}

func (l *LocationFetcher) parseAirportLocation(str string) (map[string]interface{}, error) {
	var code string

	if strings.Contains(str, "(") {
		str = strings.TrimSpace(str)
		strs := strings.Split(str, "(")
		code = strs[len(strs)-1]
		code = strings.Replace(code, ")", "", 1)
	} else {
		code = str
	}

	loc, err := l.maze.Airport(code)
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

func (l *LocationFetcher) parseLocation(str string) (map[string]interface{}, error) {
	var (
		location *maze.Location
		err      error
	)

	if strings.HasPrefix(str, "geo:") {
		location, err = l.maze.ReverseGeoURI(l.language, str)
	} else if strings.HasPrefix(str, "/") {
		_, err = l.fs.GetEntry(str)
		return nil, err
	} else {
		location, err = l.maze.Search(l.language, str)
	}

	if err != nil {
		return nil, err
	}

	return locationToMultiformat(location), nil
}

func locationToMultiformat(loc *maze.Location) map[string]interface{} {
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
