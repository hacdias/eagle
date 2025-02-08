package server

import "go.hacdias.com/indielib/micropub"

func getRequestSlug(req *micropub.Request) string {
	slug := ""
	if slugs, ok := req.Commands["slug"]; ok {
		if len(slugs) == 1 {
			slug, _ = slugs[0].(string)
		}
	}
	return slug
}

func getRequestStrings(req *micropub.Request, field string) ([]string, bool) {
	if values, ok := req.Commands[field]; ok {
		var results []string
		for _, vv := range values {
			if v, ok := vv.(string); ok {
				results = append(results, v)
			}
		}
		return results, true
	} else {
		return nil, false
	}
}

func getRequestChannels(req *micropub.Request) ([]string, bool) {
	return getRequestStrings(req, "channel")
}

func getRequestSyndicateTo(req *micropub.Request) ([]string, bool) {
	return getRequestStrings(req, "syndicate-to")
}
