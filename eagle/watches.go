package eagle

import (
	"sort"
	"time"
)

type Watch struct {
	Name   string    `yaml:"name,omitempty"`
	Author string    `yaml:"author,omitempty"`
	Season int       `yaml:"season,omitempty"`
	Rating int       `yaml:"rating,omitempty"`
	Date   time.Time `yaml:"date,omitempty"`
}

type Watches []Watch

type WatchesByYear struct {
	Years []int
	Map   map[int]Watches
}

func (w Watches) ByYear() *WatchesByYear {
	years := []int{}
	byYear := map[int]Watches{}

	for _, r := range w {
		year := r.Date.Year()

		_, ok := byYear[year]
		if !ok {
			years = append(years, year)
			byYear[year] = Watches{}
		}

		byYear[year] = append(byYear[year], r)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	for _, year := range years {
		byYear[year].Sort()
	}

	return &WatchesByYear{
		Years: years,
		Map:   byYear,
	}
}

func (w Watches) Sort() {
	sort.SliceStable(w, func(i, j int) bool {
		if w[i].Date.Equal(w[j].Date) {
			return w[i].Name < w[j].Name
		}

		return w[i].Date.After(w[j].Date)
	})
}
