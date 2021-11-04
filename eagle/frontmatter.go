package eagle

import (
	"fmt"
	"time"

	"github.com/hacdias/eagle/v2/pkg/mf2"
	"github.com/spf13/cast"
	yaml "gopkg.in/yaml.v2"
)

type Frontmatter struct {
	entry *mf2.FlatHelper

	Title          string    `yaml:"title,omitempty"`
	Description    string    `yaml:"description,omitempty"`
	Draft          bool      `yaml:"draft,omitempty"`
	Deleted        bool      `yaml:"deleted,omitempty"`
	Private        bool      `yaml:"private,omitempty"`
	NoInteractions bool      `yaml:"noInteractions,omitempty"`
	Emoji          string    `yaml:"emoji,omitempty"`
	Published      time.Time `yaml:"published,omitempty"`
	Updated        time.Time `yaml:"updated,omitempty"`
	Section        string    `yaml:"section,omitempty"`

	// MF2 flattened properties.
	Properties map[string]interface{} `yaml:"properties,omitempty"`
}

func unmarshalFrontmatter(data []byte) (*Frontmatter, error) {
	f := &Frontmatter{}
	err := yaml.Unmarshal(data, &f)
	if err != nil {
		return nil, err
	}

	// To support boolean keys, the YAML package unmarshals maps to
	// map[interface{}]interface{}. Here we recurse through the result
	// and change all maps to map[string]interface{} like we would've
	// gotten from `json`.
	//
	// Code taken from:
	// - https://github.com/gohugoio/hugo/blob/49972d0/parser/metadecoders/decoder.go
	if f.Properties != nil {
		if mm, changed := stringifyMapKeys(f.Properties); changed {
			f.Properties = mm.(map[string]interface{})
		}
	}

	return f, nil
}

// stringifyMapKeys recurses into in and changes all instances of
// map[interface{}]interface{} to map[string]interface{}. This is useful to
// work around the impedance mismatch between JSON and YAML unmarshaling that's
// described here: https://github.com/go-yaml/yaml/issues/139
//
// Inspired by https://github.com/stripe/stripe-mock, MIT licensed
func stringifyMapKeys(in interface{}) (interface{}, bool) {
	switch in := in.(type) {
	case []interface{}:
		for i, v := range in {
			if vv, replaced := stringifyMapKeys(v); replaced {
				in[i] = vv
			}
		}
	case map[string]interface{}:
		for k, v := range in {
			if vv, changed := stringifyMapKeys(v); changed {
				in[k] = vv
			}
		}
	case map[interface{}]interface{}:
		res := make(map[string]interface{})
		var (
			ok  bool
			err error
		)
		for k, v := range in {
			var ks string

			if ks, ok = k.(string); !ok {
				ks, err = cast.ToStringE(k)
				if err != nil {
					ks = fmt.Sprintf("%v", k)
				}
			}
			if vv, replaced := stringifyMapKeys(v); replaced {
				res[ks] = vv
			} else {
				res[ks] = v
			}
		}
		return res, true
	}

	return nil, false
}
