package eagle

type WebFinger struct {
	Subject string          `json:"subject"`
	Aliases []string        `json:"aliases,omitempty"`
	Links   []WebFingerLink `json:"links,omitempty"`
}

type WebFingerLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel,omitempty"`
	Type string `json:"type,omitempty"`
}
