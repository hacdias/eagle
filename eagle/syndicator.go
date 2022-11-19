package eagle

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/samber/lo"
)

type Syndicator interface {
	// Add context.Context to syndicate
	Syndicate(entry *Entry) (url string, err error)
	IsByContext(entry *Entry) bool
	Name() string
	Identifier() string
}

type SyndicatorConfig struct {
	UID  string
	Name string
}

// TODO: rename?
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

func (m *Manager) Syndicate(e *Entry, syndicators []string) ([]string, error) {
	// TODO: support syndicating deleted, especially if the post was published
	// and had a syndication. Then, we should support deleting the syndication.
	if e.Draft || e.Deleted || e.Visibility() == VisibilityPrivate {
		return nil, nil
	}

	// TODO: detect that this is a reply/like/repost to a post on my own
	// website. If so, fetch the syndications to syndicate the replies directly
	// there. For example, if I reply to a post on my website that is syndicated
	// on Twitter, I will want to syndicate that on Twitter. For now, I have to
	// directly reply to the Twitter version.
	for id, syndicator := range m.syndicators {
		if syndicator.IsByContext(e) {
			syndicators = append(syndicators, id)
		}
	}

	syndicators = lo.Uniq(syndicators)

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

		url, err := syndicator.Syndicate(e)
		if err != nil {
			errors = multierror.Append(errors, err)
		} else {
			syndications = append(syndications, url)
		}
	}

	return syndications, errors.ErrorOrNil()
}

func (m *Manager) Config() []*SyndicatorConfig {
	cfg := []*SyndicatorConfig{}

	for _, syndicator := range m.syndicators {
		cfg = append(cfg, &SyndicatorConfig{
			UID:  syndicator.Identifier(),
			Name: syndicator.Name(),
		})
	}

	return cfg
}
