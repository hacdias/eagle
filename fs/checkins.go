package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/maze"
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

// ClosestCheckin returns the closest checkin to t (less than 1 hour) and loc
// (less than 500 meters).
func (f *FS) ClosestCheckin(t time.Time, loc *maze.Location) (*eagle.Checkin, error) {
	checkins, err := f.GetCheckins(t.Year(), t.Month())
	if err != nil {
		return nil, err
	}

	// Get last month's checkins if we're close to midnight of the day before.
	if t.Day() == 1 && t.Hour() <= 5 {
		ot := t.AddDate(0, -1, 0)
		oldCheckins, err := f.GetCheckins(ot.Year(), ot.Month())
		if err != nil {
			return nil, err
		}
		checkins = append(checkins, oldCheckins...).Sort()
	}

	j := -1
	for i, c := range checkins {
		if c.Date.Before(t) {
			j = i
		} else {
			break
		}
	}

	if j == -1 {
		return nil, nil
	}

	checkin := checkins[j]

	if t.Sub(checkin.Date).Hours() > 1 {
		return nil, nil
	}

	if loc != nil {
		// If the distance is over 500 meters, ignore.
		if loc.Distance(&checkin.Location) > 500 {
			return nil, nil
		}
	}

	return checkin, nil
}
