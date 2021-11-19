package loctools

import (
	"net/http"

	gogeouri "git.jlel.se/jlelse/go-geouri"
)

type LocTools struct {
	httpClient *http.Client
}

func NewLocTools(client *http.Client) *LocTools {
	if client == nil {
		client = &http.Client{}
	}

	return &LocTools{
		httpClient: client,
	}
}

func (l *LocTools) Reverse(lang string, lon, lat float64) (map[string]interface{}, error) {
	return l.photonReverse(lang, lon, lat)
}

func (l *LocTools) FromGeoURI(lang, geouri string) (map[string]interface{}, error) {
	geo, err := gogeouri.Parse(geouri)
	if err != nil {
		return nil, err
	}

	return l.Reverse(lang, geo.Longitude, geo.Latitude)
}

func (l *LocTools) Search(lang, query string) (map[string]interface{}, error) {
	return l.photonSearch(lang, query)
}

func (l *LocTools) Airport(query string) (map[string]interface{}, error) {
	return l.aviowikiSearch(query)
}
