package maze

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
	ICAO        string              `json:"icao"`
	IATA        string              `json:"iata"`
	Name        string              `json:"name"`
	Coordinates aviowikiCoordinates `json:"coordinates"`
	Country     aviowikiCountry     `json:"country"`
	City        string              `json:"servedCity"`
}

type aviowikiCoordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type aviowikiCountry struct {
	Name string `json:"name"`
}

func (l *Maze) aviowikiSearch(query string) (*Location, error) {
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

	f := avioRes.Content[0]

	loc := &Location{
		Name:      query,
		Latitude:  f.Coordinates.Latitude,
		Longitude: f.Coordinates.Longitude,
		Country:   f.Country.Name,
		Locality:  strings.TrimSpace(strings.Split(f.City, ",")[0]),
	}

	return loc, nil
}
