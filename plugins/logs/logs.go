package logs

import (
	"bytes"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"time"

	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

var (
	_ server.MCPPlugin = &Logs{}
)

func init() {
	server.RegisterPlugin("logs", NewLogs)
}

const (
	moviesFile       = "movies.json"
	showsFile        = "shows.json"
	readingsFile     = "readings.json"
	performancesFile = "performances.json"
)

type logsConfig struct {
	Directory string
}

type Logs struct {
	core      *core.Core
	directory string
}

func NewLogs(co *core.Core, config logsConfig) (server.Plugin, error) {
	if config.Directory == "" {
		return nil, errors.New("logs directory missing")
	}

	return &Logs{
		core:      co,
		directory: config.Directory,
	}, nil
}

type logEntry interface {
	date() time.Time
}

func compareLogs[T logEntry](a, b T) int {
	// Newest first. Undated entries have the zero time, so they come last.
	return b.date().Compare(a.date())
}

type MovieLogEntry struct {
	Name     string    `json:"name"`
	Date     time.Time `json:"date,omitzero"`
	Rating   int       `json:"rating,omitempty"`
	Location string    `json:"location,omitempty"`
	Room     string    `json:"room,omitempty"`
	Row      string    `json:"row,omitempty"`
	Seat     string    `json:"seat,omitempty"`
}

func (m MovieLogEntry) date() time.Time {
	return m.Date
}

type PerformanceLogEntry struct {
	Name          string    `json:"name,omitempty"`
	Author        string    `json:"author,omitempty"`
	Rating        int       `json:"rating"`
	Date          time.Time `json:"date"`
	Location      string    `json:"location"`
	Category      string    `json:"category"`
	SupportingAct []string  `json:"supportingAct,omitempty"`
}

func (p PerformanceLogEntry) date() time.Time {
	return p.Date
}

type ReadingLogEntry struct {
	Name      string    `json:"name"`
	Author    string    `json:"author"`
	Rating    int       `json:"rating,omitempty"`
	Publisher string    `json:"publisher,omitempty"`
	Pages     int       `json:"pages,omitempty"`
	Date      time.Time `json:"date,omitzero"`
	UID       string    `json:"uid"`
}

func (r ReadingLogEntry) date() time.Time {
	return r.Date
}

type ShowLogEntry struct {
	Name   string    `json:"name"`
	Season int       `json:"season,omitempty"`
	Date   time.Time `json:"date,omitzero"`
	Rating int       `json:"rating,omitempty"`
}

func (s ShowLogEntry) date() time.Time {
	return s.Date
}

// path returns the path of the given log file in the source directory.
func (l *Logs) path(file string) string {
	return path.Join("/", l.directory, file)
}

func (l *Logs) get[T any](file string) ([]T, error) {
	logs := []T{}
	err := l.core.ReadJSON(l.path(file), &logs)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (l *Logs) add[T logEntry](file string, entry T) error {
	logs, err := l.get[T](file)
	if err != nil {
		return err
	}

	logs = append([]T{entry}, logs...)
	return l.write(file, logs)
}

// edit replaces the single entry for which match returns true.
func (l *Logs) edit[T logEntry](file string, match func(T) bool, entry T) error {
	logs, err := l.get[T](file)
	if err != nil {
		return err
	}

	idx := -1
	for i, log := range logs {
		if match(log) {
			if idx != -1 {
				return errors.New("multiple log entries match")
			}
			idx = i
		}
	}

	if idx == -1 {
		return errors.New("no log entry matches")
	}

	logs[idx] = entry
	return l.write(file, logs)
}

func (l *Logs) write[T logEntry](file string, logs []T) error {
	data, err := marshalLogs(logs)
	if err != nil {
		return err
	}

	filename := l.path(file)
	return l.core.WriteFile(filename, data, "data: update "+filename)
}

func marshalLogs[T logEntry](logs []T) ([]byte, error) {
	slices.SortStableFunc(logs, compareLogs[T])

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(logs)
	return buf.Bytes(), err
}
