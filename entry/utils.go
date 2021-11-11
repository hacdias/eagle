package entry

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/thoas/go-funk"
)

var allowedLetters = []rune("abcdefghijklmnopqrstuvwxyz")

func newID(slug string, t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}

	if slug == "" {
		slug = funk.RandomString(5, allowedLetters)
	}

	return fmt.Sprintf("/%04d/%02d/%02d/%s", t.Year(), t.Month(), t.Day(), slug)
}

func cleanID(id string) string {
	id = path.Clean(id)
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")
	return "/" + id
}
