package maze

import (
	"math"
	"net/http"

	gogeouri "git.jlel.se/jlelse/go-geouri"
)

type Location struct {
	Latitude  float64 `json:"latitude,omitempty" yaml:"latitude,omitempty" csv:"latitude"`
	Longitude float64 `json:"longitude,omitempty" yaml:"longitude,omitempty" csv:"longitude"`
	Name      string  `json:"name,omitempty" yaml:"name,omitempty" csv:"name"`
	Locality  string  `json:"locality,omitempty" yaml:"locality,omitempty" csv:"locality"`
	Region    string  `json:"region,omitempty" yaml:"region,omitempty" csv:"region"`
	Country   string  `json:"country,omitempty" yaml:"country,omitempty" csv:"country"`
}

// Distance returns the distance, in meters, between l1 and l2.
func (l1 *Location) Distance(l2 *Location) float64 {
	if l2 == nil || l1 == nil {
		return 0
	}

	lat1 := l1.Latitude * (math.Pi / 180)
	lon1 := l1.Longitude * (math.Pi / 180)
	lat2 := l2.Latitude * (math.Pi / 180)
	lon2 := l2.Longitude * (math.Pi / 180)

	dlon := lon2 - lon1
	dlat := lat2 - lat1

	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	c := 2 * math.Asin(math.Sqrt(a))
	r := float64(6371)

	return c * r * 1000
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
