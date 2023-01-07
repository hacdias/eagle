package eagle

import (
	"sort"
	"time"

	"github.com/hacdias/eagle/pkg/maze"
)

type Checkin struct {
	Date time.Time `csv:"date"`
	maze.Location
}

type Checkins []*Checkin

func (cc Checkins) Sort() Checkins {
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].Date.Before(cc[j].Date)
	})
	return cc
}
