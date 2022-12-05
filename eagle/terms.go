package eagle

import "sort"

type Terms []string

type GroupedTerms struct {
	Groups []string
	Terms  map[string]Terms
}

func (t Terms) GroupByFirstChar() *GroupedTerms {
	chars := []string{}
	byChar := map[string]Terms{}

	for _, term := range t {
		char := string([]rune(term)[0])

		_, ok := byChar[char]
		if !ok {
			chars = append(chars, char)
			byChar[char] = []string{}
		}

		byChar[char] = append(byChar[char], term)
	}

	sort.Strings(chars)

	for _, char := range chars {
		sort.Strings(byChar[char])
	}

	return &GroupedTerms{
		Groups: chars,
		Terms:  byChar,
	}
}
