package eagle

import "sort"

type Term struct {
	Name  string
	Count int
}

type Terms []*Term

func (t Terms) SortByName(desc bool) Terms {
	if desc {
		sort.SliceStable(t, func(i, j int) bool {
			return t[i].Name > t[j].Name
		})
	} else {
		sort.SliceStable(t, func(i, j int) bool {
			return t[i].Name < t[j].Name
		})
	}

	return t
}

func (t Terms) SortByCount(desc bool) Terms {
	if desc {
		sort.SliceStable(t, func(i, j int) bool {
			return t[i].Count > t[j].Count
		})
	} else {
		sort.SliceStable(t, func(i, j int) bool {
			return t[i].Count < t[j].Count
		})
	}

	return t
}

type GroupedTerms struct {
	Groups []string
	Terms  map[string]Terms
}

func (t Terms) GroupByFirstChar() *GroupedTerms {
	chars := []string{}
	byChar := map[string]Terms{}

	for _, term := range t {
		char := string([]rune(term.Name)[0])

		_, ok := byChar[char]
		if !ok {
			chars = append(chars, char)
			byChar[char] = []*Term{}
		}

		byChar[char] = append(byChar[char], term)
	}

	sort.Strings(chars)

	for _, char := range chars {
		byChar[char].SortByName(false)
	}

	return &GroupedTerms{
		Groups: chars,
		Terms:  byChar,
	}
}
