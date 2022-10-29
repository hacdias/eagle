package entry

import (
	"fmt"
	"math/rand"
	"path"
	"strings"
	"time"
)

var allowedLetters = []rune("abcdefghijklmnopqrstuvwxyz")

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func NewSlug() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = allowedLetters[seededRand.Intn(len(allowedLetters))]
	}

	return string(b)
}

func NewTimeSlug(t time.Time) string {
	ns := t.Nanosecond()
	for ns > 99 {
		ns /= 10
	}
	return fmt.Sprintf("%02dh%02dm%02ds%02d", t.Hour(), t.Minute(), t.Second(), ns)
}

func NewID(slug string, t time.Time) string {
	if t.IsZero() {
		t = time.Now().Local()
	}

	if slug == "" {
		slug = NewTimeSlug(t)
	}

	return fmt.Sprintf("/%04d/%02d/%02d/%s", t.Year(), t.Month(), t.Day(), slug)
}

func cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}
