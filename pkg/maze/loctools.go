package maze

import (
	"net/http"

	gogeouri "git.jlel.se/jlelse/go-geouri"
)

type Location struct {
	Latitude  float64 `json:"latitude,omitempty" yaml:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty" yaml:"longitude,omitempty"`
	Name      string  `json:"name,omitempty" yaml:"name,omitempty"`
	Locality  string  `json:"locality,omitempty" yaml:"locality,omitempty"`
	Region    string  `json:"region,omitempty" yaml:"region,omitempty"`
	Country   string  `json:"country-name,omitempty" yaml:"country-name,omitempty"`
}

type Maze struct {
	httpClient *http.Client
}

func NewMaze(client *http.Client) *Maze {
	if client == nil {
		client = &http.Client{}
	}

	return &Maze{
		httpClient: client,
	}
}

func (l *Maze) Reverse(lang string, lon, lat float64) (*Location, error) {
	return l.photonReverse(lang, lon, lat)
}

func (l *Maze) ReverseGeoURI(lang, geoUri string) (*Location, error) {
	geo, err := gogeouri.Parse(geoUri)
	if err != nil {
		return nil, err
	}

	return l.Reverse(lang, geo.Longitude, geo.Latitude)
}

func (l *Maze) Search(lang, query string) (*Location, error) {
	return l.photonSearch(lang, query)
}

func (l *Maze) Airport(query string) (*Location, error) {
	return l.aviowikiSearch(query)
}
