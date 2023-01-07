package eagle

import (
	"sort"
	"time"
)

type Checkin struct {
	Date      time.Time `csv:"date"`
	Latitude  float64   `csv:"latitude"`
	Longitude float64   `csv:"longitude"`
	Name      string    `csv:"name"`
	Locality  string    `csv:"locality"`
	Region    string    `csv:"region"`
	Country   string    `csv:"country"`
}

func (c *Checkin) Multiformats() map[string]interface{} {
	props := map[string]interface{}{
		"latitude":  c.Latitude,
		"longitude": c.Longitude,
	}

	if c.Name != "" {
		props["name"] = c.Name
	}

	if c.Locality != "" {
		props["locality"] = c.Locality
	}

	if c.Region != "" {
		props["region"] = c.Region
	}

	if c.Country != "" {
		props["country-name"] = c.Country
	}

	return map[string]interface{}{
		"type":       "h-adr",
		"properties": props,
	}
}

type Checkins []*Checkin

func (cc Checkins) Sort() Checkins {
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].Date.Before(cc[j].Date)
	})
	return cc
}
