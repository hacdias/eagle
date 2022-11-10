package eagle

import "github.com/hacdias/eagle/v4/pkg/xray"

type Sidecar struct {
	Targets      []string     `json:"targets,omitempty"`
	Context      *xray.Post   `json:"context,omitempty"`
	Replies      []*xray.Post `json:"replies,omitempty"`
	Interactions []*xray.Post `json:"interactions,omitempty"`
}

func (s *Sidecar) MentionsCount() int {
	return len(s.Replies) + len(s.Interactions)
}
