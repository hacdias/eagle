package eagle

import (
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/thoas/go-funk"
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

func (m *Manager) Syndicate(ee *Entry, syndicators []string) ([]string, error) {
	if ee.Draft {
		return nil, errors.New("cannot syndicate draft entry")
	}

	if ee.Visibility() == VisibilityPrivate {
		return nil, errors.New("cannot syndicate private entry")
	}

	if ee.Deleted {
		return nil, errors.New("cannot syndicate deleted entry")
	}

	// TODO: detect that this is a reply/like/repost to a post on my own
	// website. If so, fetch the syndications to syndicate the replies directly
	// there. For example, if I reply to a post on my website that is syndicated
	// on Twitter, I will want to syndicate that on Twitter. For now, I have to
	// directly reply to the Twitter version.
	for id, syndicator := range m.syndicators {
		if syndicator.IsByContext(ee) {
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

		url, err := syndicator.Syndicate(ee)
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