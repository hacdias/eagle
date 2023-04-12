package eagle

import (
	"fmt"
	"time"

	"github.com/hacdias/eagle/pkg/maze"
	"github.com/spf13/cast"
	yaml "gopkg.in/yaml.v2"
)

type Listing struct {
	DisablePagination bool `yaml:"disablePagination,omitempty"`
	OrderByUpdated    bool `yaml:"orderByUpdated,omitempty"`
	ItemsPerPage      int  `yaml:"itemsPerPage,omitempty"`
}

type Context struct {
	// TODO: rename 'name' to 'author' at some point.
	Author    string    `yaml:"name,omitempty"`
	URL       string    `yaml:"url,omitempty"`
	Content   string    `yaml:"content,omitempty"`
	Published time.Time `yaml:"published,omitempty"`
}

type Read struct {
	Name      string `yaml:"name,omitempty"`
	Author    string `yaml:"author,omitempty"`
	Publisher string `yaml:"publisher,omitempty"`
	Pages     int    `yaml:"pages,omitempty"`
	UID       string `yaml:"uid,omitempty"`
}

type FrontMatter struct {
	Title              string                 `yaml:"title,omitempty"`
	Description        string                 `yaml:"description,omitempty"`
	Draft              bool                   `yaml:"draft,omitempty"`
	Unlisted           bool                   `yaml:"unlisted,omitempty"`
	Deleted            bool                   `yaml:"deleted,omitempty"`
	Published          time.Time              `yaml:"published,omitempty"`
	Updated            time.Time              `yaml:"updated,omitempty"`
	Sections           []string               `yaml:"section,omitempty"`
	Template           string                 `yaml:"template,omitempty"`
	NoSendInteractions bool                   `yaml:"noSendInteractions,omitempty"`
	CoverImage         string                 `yaml:"coverImage,omitempty"`
	Properties         map[string]interface{} `yaml:"properties,omitempty"` // "Flat" MF2 Properties.
	NoIndex            bool                   `yaml:"noIndex,omitempty"`
	Listing            *Listing               `yaml:"listing,omitempty"`
	Taxonomies         map[string][]string    `yaml:"taxonomies,omitempty"`
	Location           *maze.Location         `yaml:"location,omitempty"`
	Context            *Context               `yaml:"context,omitempty"`
}

func (f *FrontMatter) Taxonomy(name string) []string {
	if v, ok := f.Taxonomies[name]; ok {
		return v
	}

	return []string{}
}

func unmarshalFrontMatter(data []byte) (*FrontMatter, error) {
	f := &FrontMatter{}
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
