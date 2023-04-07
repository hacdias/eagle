package eagle

import "github.com/hacdias/eagle/pkg/xray"

type Mention struct {
	xray.Post
	Hidden bool   `json:"hidden,omitempty"`
	ID     string `json:"id,omitempty"`
}

type Sidecar struct {
	Context *xray.Post `json:"context,omitempty"`
}

func (s *Sidecar) Empty() bool {
	return s.Context == nil
}
