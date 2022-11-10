package eagle

import (
	"sort"
	"time"
)

type Read struct {
	ID     string    `json:"id"`
	Date   time.Time `json:"date"`
	Name   string    `json:"name"`
	Author string    `json:"author"`
}

type ReadList []*Read

type ReadsSummary struct {
	ToRead   ReadList    `json:"to-read"`
	Reading  ReadList    `json:"reading"`
	Finished ReadsByYear `json:"finished"`
}

type ReadsByYear struct {
	Years []int
	Map   map[int]ReadList
}

func (rd ReadList) ByYear() *ReadsByYear {
	years := []int{}
	byYear := map[int]ReadList{}

	for _, r := range rd {
		year := r.Date.Year()

		_, ok := byYear[year]
		if !ok {
			years = append(years, year)
			byYear[year] = ReadList{}
		}

		byYear[year] = append(byYear[year], r)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	for _, year := range years {
		byYear[year].SortByName()
	}

	return &ReadsByYear{
		Years: years,
		Map:   byYear,
	}
}

func (rd ReadList) SortByName() {
	sort.SliceStable(rd, func(i, j int) bool {
		return rd[i].Name < rd[j].Name
	})
}
