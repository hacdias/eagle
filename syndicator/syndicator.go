package syndicator

import (
	"fmt"

	"github.com/hacdias/eagle/v2/entry"
	"github.com/hashicorp/go-multierror"
	"github.com/thoas/go-funk"
)

type Syndicator interface {
	Syndicate(entry *entry.Entry) (url string, err error)
	IsByContext(entry *entry.Entry) bool
	Name() string
	Identifier() string
}

type Manager struct {
	syndicators map[string]Syndicator
}

func NewManager() *Manager {
	return &Manager{
		syndicators: map[string]Syndicator{},
	}
}

func (m *Manager) Add(s Syndicator) {
	m.syndicators[s.Identifier()] = s
}

func (m *Manager) Syndicate(entry *entry.Entry, syndicators []string) ([]string, error) {
	// TODO(future): detect that this is a reply/like/repost to a post on my own
	// website. If so, fetch the syndications to syndicate the replies directly
	// there. For example, if I reply to a post on my website that is syndicated
	// on Twitter, I will want to syndicate that on Twitter. For now, I have to
	// directly reply to the Twitter version.

	for id, syndicator := range m.syndicators {
		if syndicator.IsByContext(entry) {
			syndicators = append(syndicators, id)
		}
	}

	syndicators = funk.UniqString(syndicators)

	var (
		errors       *multierror.Error
		syndications []string
	)

	for _, id := range syndicators {
		syndicator, ok := m.syndicators[id]
		if !ok {
			errors = multierror.Append(errors, fmt.Errorf("unknown syndication service: %s", id))
			continue
		}

		url, err := syndicator.Syndicate(entry)
		if err != nil {
			errors = multierror.Append(errors, err)
		} else {
			syndications = append(syndications, url)
		}
	}

	return syndications, errors.ErrorOrNil()
}
