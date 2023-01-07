package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/hacdias/eagle/eagle"
)

func checkinsFilename(year int, month time.Month) string {
	csvName := fmt.Sprintf("%04d-%02d.csv", year, month)
	filename := filepath.Join("data", "checkins", csvName)
	return filename
}

func (f *FS) SaveCheckin(c *eagle.Checkin) error {
	f.checkinsMu.Lock()
	defer f.checkinsMu.Unlock()

	year := c.Date.Year()
	month := c.Date.Month()

	checkins, err := f.GetCheckins(year, month)
	if err != nil {
		return err
	}
	checkins = append(checkins, c).Sort()

	data, err := gocsv.MarshalBytes(&checkins)
	if err != nil {
		return err
	}

	filename := checkinsFilename(year, month)
	err = f.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return err
	}

	return f.WriteFile(filename, data)
}

func (f *FS) GetCheckins(year int, month time.Month) (eagle.Checkins, error) {
	filename := checkinsFilename(year, month)
	checkins := []*eagle.Checkin{}

	file, err := f.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return checkins, nil
		}

		return nil, err
	}

	err = gocsv.Unmarshal(file, &checkins)
	if err != nil {
		return nil, err
	}

	return checkins, err
}

func (f *FS) ClosestCheckin(t time.Time) (*eagle.Checkin, error) {
	checkins, err := f.GetCheckins(t.Year(), t.Month())
	if err != nil {
		return nil, err
	}

	if t.Day() == 1 {
		lastMonth := t.AddDate(0, -1, 0)

		lastMonthCheckins, err := f.GetCheckins(lastMonth.Year(), lastMonth.Month())
		if err != nil {
			return nil, err
		}

		checkins = append(checkins, lastMonthCheckins...).Sort()
	}

	for i, c := range checkins {
		if c.Date.After(t) {
			if i != 0 {
				return checkins[i-1], nil
			} else {
				return nil, nil
			}
		}
	}

	return nil, errors.New("not implemented")
}
