package core

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

const (
	moviesLogPath       = "/data/logs/movies.json"
	showsLogPath        = "/data/logs/shows.json"
	readingsLogPath     = "/data/logs/readings.json"
	performancesLogPath = "/data/logs/performances.json"
)

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

func (co *Core) getLogs[T any](path string) ([]T, error) {
	logs := []T{}
	err := co.ReadJSON(path, &logs)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (co *Core) GetReadingLogs() ([]ReadingLogEntry, error) {
	return co.getLogs[ReadingLogEntry](readingsLogPath)
}

func (co *Core) GetMovieLogs() ([]MovieLogEntry, error) {
	return co.getLogs[MovieLogEntry](moviesLogPath)
}

func (co *Core) GetShowLogs() ([]ShowLogEntry, error) {
	return co.getLogs[ShowLogEntry](showsLogPath)
}

func (co *Core) GetPerformanceLogs() ([]PerformanceLogEntry, error) {
	return co.getLogs[PerformanceLogEntry](performancesLogPath)
}

func (co *Core) addLog[T logEntry](path, message string, entry T) error {
	logs, err := co.getLogs[T](path)
	if err != nil {
		return err
	}

	logs = append([]T{entry}, logs...)
	return co.writeLogs(path, message, logs)
}

// editLog replaces the single entry for which match returns true.
func (co *Core) editLog[T logEntry](path, message string, match func(T) bool, entry T) error {
	logs, err := co.getLogs[T](path)
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
	return co.writeLogs(path, message, logs)
}

func (co *Core) writeLogs[T logEntry](path, message string, logs []T) error {
	data, err := marshalLogs(logs)
	if err != nil {
		return err
	}

	return co.WriteFile(path, data, message)
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

func (co *Core) AddReadingLog(entry ReadingLogEntry) error {
	return co.addLog(readingsLogPath, fmt.Sprintf("data: add %s to readings log", entry.Name), entry)
}

func (co *Core) AddMovieLog(entry MovieLogEntry) error {
	return co.addLog(moviesLogPath, fmt.Sprintf("data: add %s to movies log", entry.Name), entry)
}

func (co *Core) AddShowLog(entry ShowLogEntry) error {
	return co.addLog(showsLogPath, fmt.Sprintf("data: add %s to shows log", entry.Name), entry)
}

func (co *Core) AddPerformanceLog(entry PerformanceLogEntry) error {
	return co.addLog(performancesLogPath, fmt.Sprintf("data: add %s to performances log", cmp.Or(entry.Name, entry.Author)), entry)
}

func (co *Core) EditReadingLog(date time.Time, name string, entry ReadingLogEntry) error {
	return co.editLog(readingsLogPath, fmt.Sprintf("data: edit %s in readings log", entry.Name), func(r ReadingLogEntry) bool {
		return r.Date.Equal(date) && r.Name == name
	}, entry)
}

func (co *Core) EditMovieLog(date time.Time, name string, entry MovieLogEntry) error {
	return co.editLog(moviesLogPath, fmt.Sprintf("data: edit %s in movies log", entry.Name), func(m MovieLogEntry) bool {
		return m.Date.Equal(date) && m.Name == name
	}, entry)
}

func (co *Core) EditShowLog(date time.Time, name string, season int, entry ShowLogEntry) error {
	return co.editLog(showsLogPath, fmt.Sprintf("data: edit %s in shows log", entry.Name), func(s ShowLogEntry) bool {
		return s.Date.Equal(date) && s.Name == name && s.Season == season
	}, entry)
}

// EditPerformanceLog edits the performance on the given date that matches the
// given name, author, or both. Empty values are not compared, but at least one
// of them must be given.
func (co *Core) EditPerformanceLog(date time.Time, name, author string, entry PerformanceLogEntry) error {
	if name == "" && author == "" {
		return errors.New("name or author is required to identify the performance")
	}

	return co.editLog(performancesLogPath, fmt.Sprintf("data: edit %s in performances log", cmp.Or(entry.Name, entry.Author)), func(p PerformanceLogEntry) bool {
		return p.Date.Equal(date) && (name == "" || p.Name == name) && (author == "" || p.Author == author)
	}, entry)
}
