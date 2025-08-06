package miniflux

import "encoding/xml"

// Specification: https://opml.org/spec2.opml

type opmlFile struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    opmlHead `xml:"head"`
	Body    opmlBody `xml:"body"`
}

type opmlHead struct {
	Title string `xml:"title"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Title    string        `xml:"title,attr,omitempty"`
	XMLUrl   string        `xml:"xmlUrl,attr,omitempty"`
	Text     string        `xml:"text,attr,omitempty"`
	HTMLUrl  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutline `xml:"outline,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
}

func makeOpml(feeds map[string][]feed) ([]byte, error) {
	opmlData := opmlFile{
		Version: "2.0",
		Head: opmlHead{
			Title: "Blogroll",
		},
	}

	for category, feedsList := range feeds {
		categoryOutline := opmlOutline{
			Text: category,
		}

		for _, f := range feedsList {
			categoryOutline.Outlines = append(categoryOutline.Outlines, opmlOutline{
				Text:    f.Title,
				XMLUrl:  f.Feed,
				HTMLUrl: f.Site,
				Type:    "rss",
			})
		}

		opmlData.Body.Outlines = append(opmlData.Body.Outlines, categoryOutline)
	}

	bytes, err := xml.MarshalIndent(opmlData, "", "  ")
	if err != nil {
		return nil, err
	}
	bytes = append([]byte(xml.Header), bytes...)
	return bytes, nil
}
