package eagle

import "time"

type Checkin struct {
	Date      time.Time `csv:"date"`
	Latitude  float64   `csv:"latitude"`
	Longitude float64   `csv:"longitude"`
	Name      string    `csv:"name"`
	Locality  string    `csv:"locality"`
	Region    string    `csv:"region"`
	Country   string    `csv:"country"`
}
