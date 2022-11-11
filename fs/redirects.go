package fs

import (
	"fmt"
	"os"
	"strings"
)

const (
	RedirectsFile = "redirects"
)

func (fs *FS) AppendRedirect(old, new string) error {
	f, err := fs.OpenFile(RedirectsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s %s\n", old, new))
	return err
}

func (fs *FS) LoadRedirects(ignoreMalformed bool) (map[string]string, error) {
	redirects := map[string]string{}

	data, err := fs.ReadFile(RedirectsFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) == 2 {
			redirects[parts[0]] = parts[1]
		} else if !ignoreMalformed {
			return nil, fmt.Errorf("found invalid redirect entry: %s", line)
		}
	}

	return redirects, nil
}
