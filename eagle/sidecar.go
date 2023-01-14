package eagle

import "github.com/hacdias/eagle/pkg/xray"

type Mention struct {
	xray.Post
	Hidden bool   `json:"hidden,omitempty"`
	ID     string `json:"id,omitempty"`
}

type Sidecar struct {
	Targets      []string   `json:"targets,omitempty"`
	Context      *xray.Post `json:"context,omitempty"`
	Replies      []*Mention `json:"replies,omitempty"`
	Interactions []*Mention `json:"interactions,omitempty"`
}

func (s *Sidecar) MentionsCount() int {
	return len(s.Replies) + len(s.Interactions)
}

func (s *Sidecar) Empty() bool {
	return len(s.Targets) == 0 &&
		s.Context == nil &&
		len(s.Replies) == 0 &&
		len(s.Interactions) == 0
}
