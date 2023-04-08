package eagle

import "github.com/hacdias/eagle/pkg/xray"

type Sidecar struct {
	Context *xray.Post `json:"context,omitempty"`
}

func (s *Sidecar) Empty() bool {
	return s.Context == nil
}
