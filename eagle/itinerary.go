package eagle

import (
	"github.com/hacdias/eagle/v2/entry"
	"github.com/karlseguin/typed"
)

func (e *Eagle) processItinerary(ee *entry.Entry) error {
	mm := ee.Helper()

	var legs []typed.Typed

	if leg, ok := mm.Properties.ObjectIf(mm.TypeProperty()); ok {
		legs = append(legs, leg)
	} else if llegs, ok := mm.Properties.ObjectsIf(mm.TypeProperty()); ok {
		legs = llegs
	} else {
		return nil
	}

	if len(legs) == 0 {
		return nil
	}

	var lastDest map[string]interface{}

	for _, leg := range legs {
		props, ok := leg.ObjectIf("properties")
		if !ok {
			continue
		}

		transitType := props.String("transit-type")

		if origin := props.String("origin"); origin != "" {
			loc, err := e.parseLocation(origin, transitType == "air")
			if err != nil {
				return err
			}
			props["origin"] = loc
		}

		if destination := props.String("destination"); destination != "" {
			loc, err := e.parseLocation(destination, transitType == "air")
			if err != nil {
				return err
			}
			props["destination"] = loc
			lastDest = loc
		}
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

	return err
}
