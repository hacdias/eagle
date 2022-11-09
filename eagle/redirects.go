package eagle

import "strings"

func (e *Eagle) GetRedirects() map[string]string {
	return e.redirects
}

func (e *Eagle) initRedirects() error {
	redirects := map[string]string{}

	data, err := e.FS.ReadFile("redirects")
	if err != nil {
		return err
	}

	strs := strings.Split(string(data), "\n")

	for _, str := range strs {
		if strings.TrimSpace(str) == "" {
			continue
		}

		parts := strings.Split(str, " ")
		if len(parts) != 2 {
			e.log.Warnf("found invalid redirect entry: %s", str)
		}

		redirects[parts[0]] = parts[1]
	}

	e.redirects = redirects
	return nil
}
