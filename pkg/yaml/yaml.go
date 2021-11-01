// Drop-in replacement for yaml to convert all maps to map[string]interface{}
// instead of map[interface{}]interface{}
//
// Code taken from:
// - https://github.com/gohugoio/hugo/blob/49972d0/parser/metadecoders/decoder.go
package yaml

import (
	"fmt"

	"github.com/spf13/cast"
	yaml "gopkg.in/yaml.v2"
)

func Unmarshal(data []byte, v interface{}) error {
	err := yaml.Unmarshal(data, v)
	if err != nil {
		return err
	}

	// To support boolean keys, the YAML package unmarshals maps to
	// map[interface{}]interface{}. Here we recurse through the result
	// and change all maps to map[string]interface{} like we would've
	// gotten from `json`.
	var ptr interface{}
	switch v := v.(type) {
	case *map[string]interface{}:
		ptr = *v
	case *interface{}:
		ptr = *v
	default:
		// Not a map.
	}

	if ptr != nil {
		if mm, changed := stringifyMapKeys(ptr); changed {
			switch v := v.(type) {
			case *map[string]interface{}:
				*v = mm.(map[string]interface{})
			case *interface{}:
				*v = mm
			}
		}
	}

	return nil
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

func Marshal(in interface{}) (out []byte, err error) {
	return yaml.Marshal(in)
}
