package migrate

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/araddon/dateparse"
	"github.com/gosimple/slug"
	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v2"
)

type read struct {
	Date   time.Time `yaml:"date"`
	Author string    `yaml:"author"`
	Name   string    `yaml:"name"`
	ISBN   string    `yaml:"isbn"`
	Tags   []string  `yaml:"tags"`
}

type reads struct {
	Read             []read `yaml:"read"`
	ToRead           []read `yaml:"to-read"`
	OwnNotRead       []read `yaml:"own-not-read"`
	CurrentlyReading []read `yaml:"currently-reading"`
}

func getReads(e *eagle.Eagle) error {
	data, err := ioutil.ReadFile(filepath.Join(dataPath, "reads.yaml"))
	if err != nil {
		return err
	}

	var rr *reads
	err = yaml.Unmarshal(data, &rr)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	for _, r := range rr.CurrentlyReading {
		errs = multierror.Append(errs, convertRead(e, &r, "reading", dateparse.MustParse("2021-11-07T12:00:00Z")))
	}

	for _, r := range rr.OwnNotRead {
		errs = multierror.Append(errs, convertRead(e, &r, "to-read", dateparse.MustParse("2021-11-07T12:00:00Z")))
	}

	for _, r := range rr.ToRead {
		errs = multierror.Append(errs, convertRead(e, &r, "to-read", dateparse.MustParse("2021-11-07T12:00:00Z")))
	}

	for _, r := range rr.Read {
		errs = multierror.Append(errs, convertRead(e, &r, "finished", dateparse.MustParse("2020-12-31T12:00:00Z")))
	}

	return errs.ErrorOrNil()
}

func makeDescription(r *read, status string) string {
	str := ""

	switch status {
	case "reading":
		str = "Currently reading: "
	case "to-read":
		str = "Want to read: "
	case "finished":
		str = "Finished reading: "
	}

	str += r.Name + " by " + r.Author

	if r.ISBN != "" {
		str += ", ISBN: " + r.ISBN
	}

	return str
}

func convertRead(e *eagle.Eagle, r *read, status string, defDate time.Time) error {
	props := map[string]interface{}{
		"author": r.Author,
		"name":   r.Name,
	}

	if r.ISBN != "" {
		props["uid"] = "isbn:" + r.ISBN
	}

	entry := &entry.Entry{
		Frontmatter: entry.Frontmatter{
			Description: makeDescription(r, status),
			Published:   r.Date,
			Sections:    []string{"reads"},
			Properties: map[string]interface{}{
				"read-of": map[string]interface{}{
					"properties": props,
					"type":       "h-cite",
				},
				"read-status": status,
			},
		},
	}

	if entry.Published.IsZero() {
		entry.Published = defDate
	}

	year := entry.Published.Year()
	month := entry.Published.Month()
	day := entry.Published.Day()
	entry.ID = fmt.Sprintf("/%04d/%02d/%02d/%s", year, month, day, slug.Make(r.Name))

	return e.SaveEntry(entry)
}
