package eagle

// From https://github.com/thoas/go-funk
func uniqString(a []string) []string {
	var (
		length  = len(a)
		seen    = make(map[string]struct{}, length)
		j       = 0
		results = make([]string, 0)
	)

	for i := 0; i < length; i++ {
		v := a[i]

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		results = append(results, v)
		j++
	}

	return results
}
