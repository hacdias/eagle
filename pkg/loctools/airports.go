package loctools

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
)

type aviowikiResponse struct {
	Content []*aviowikiContent `json:"content"`
}

type aviowikiContent struct {
	ICAO        string               `json:"icao"`
	IATA        string               `json:"iata"`
	Name        string               `json:"name"`
	Coordinates *aviowikiCoordinates `json:"coordinates"`
	Country     *aviowikiCountry     `json:"country"`
	City        string               `json:"servedCity"`
}

type aviowikiCoordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type aviowikiCountry struct {
	Name string `json:"name"`
}

func (l *LocTools) aviowikiSearch(query string) (map[string]interface{}, error) {
	uv := url.Values{}
	uv.Set("query", query)
	uv.Set("size", "1")

	res, err := l.httpClient.Get("https://api.aviowiki.com/free/airports/search?" + uv.Encode())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var avioRes *aviowikiResponse
	err = json.Unmarshal(data, &avioRes)
	if err != nil {
		return nil, err
	}

	if avioRes == nil || len(avioRes.Content) == 0 {
		return nil, errors.New("no airport found")
	}

	props := map[string]interface{}{
		"name": query,
	}

	f := avioRes.Content[0]

	if f.Coordinates != nil {
		props["longitude"] = f.Coordinates.Longitude
		props["latitude"] = f.Coordinates.Latitude
	}

	if f.Country.Name != "" {
		props["country-name"] = f.Country.Name
	}

	if f.City != "" {
		props["locality"] = strings.TrimSpace(strings.Split(f.City, ",")[0])
	}

	return map[string]interface{}{
		"properties": props,
		"type":       "h-adr",
	}, nil
}
