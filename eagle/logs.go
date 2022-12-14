package eagle

import (
	"sort"
	"time"
)

type Log struct {
	Name   string    `yaml:"name,omitempty"`
	Author string    `yaml:"author,omitempty"`
	URL    string    `yaml:"url,omitempty"`
	Season int       `yaml:"season,omitempty"` // TODO: remove.
	Rating int       `yaml:"rating,omitempty"`
	Date   time.Time `yaml:"date,omitempty"`
}

type Logs []Log

type LogsByYear struct {
	Years []int
	Map   map[int]Logs
}

func (l Logs) ByYear() *LogsByYear {
	years := []int{}
	byYear := map[int]Logs{}

	for _, r := range l {
		year := r.Date.Year()

		_, ok := byYear[year]
		if !ok {
			years = append(years, year)
			byYear[year] = Logs{}
		}

		byYear[year] = append(byYear[year], r)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	for _, year := range years {
		byYear[year].Sort()
	}

	return &LogsByYear{
		Years: years,
		Map:   byYear,
	}
}

func (l Logs) Sort() {
	sort.SliceStable(l, func(i, j int) bool {
		if l[i].Date.Equal(l[j].Date) {
			return l[i].Name < l[j].Name
		}

		return l[i].Date.After(l[j].Date)
	})
}
