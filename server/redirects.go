package server

import "strings"

func (s *Server) initRedirects() error {
	redirects := map[string]string{}

	data, err := s.fs.ReadFile("redirects")
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
			s.log.Warnf("found invalid redirect entry: %s", str)
		}

		redirects[parts[0]] = parts[1]
	}

	s.redirects = redirects
	return nil
}
