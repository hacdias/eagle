package mf2

import (
	"strings"

	"github.com/karlseguin/typed"
)

type FlatHelper struct {
	Map        typed.Typed
	Properties typed.Typed

	postType     Type
	typeProperty string
}

func NewFlatHelper(data map[string]interface{}) *FlatHelper {
	if data == nil {
		return nil
	}

	return &FlatHelper{
		Map:        typed.New(data),
		Properties: typed.New(data["properties"].(map[string]interface{})),
	}
}

func (m *FlatHelper) PostType() Type {
	if m.postType == "" {
		m.postType, m.typeProperty = DiscoverType(m.Map)
	}

	return m.postType
}

func (m *FlatHelper) TypeProperty() string {
	if m.postType == "" {
		m.postType, m.typeProperty = DiscoverType(m.Map)
	}

	return m.typeProperty
}

func (m *FlatHelper) Type() string {
	return m.Map.StringOr("type", "")
}

func (m *FlatHelper) String(prop string) string {
	if v, ok := m.Properties.StringIf(prop); ok {
		return v
	}

	if v, ok := m.Properties.StringsIf(prop); ok && len(v) > 0 {
		return v[0]
	}

	return ""
}

func (m *FlatHelper) Name() string {
	return m.Properties.StringOr("name", "")
}

func (m *FlatHelper) LocalityCountry() string {
	strs := []string{}

	if v, ok := m.Properties.StringIf("locality"); ok {
		strs = append(strs, v)
	}

	if v, ok := m.Properties.StringIf("country-name"); ok {
		strs = append(strs, v)
	}

	return strings.Join(strs, ", ")
}

func (m *FlatHelper) LocalityRegionCountry() string {
	strs := []string{}

	if v, ok := m.Properties.StringIf("locality"); ok {
		strs = append(strs, v)
	}

	if v, ok := m.Properties.StringIf("region"); ok {
		strs = append(strs, v)
	}

	if v, ok := m.Properties.StringIf("country-name"); ok {
		strs = append(strs, v)
	}

	return strings.Join(strs, ", ")
}

func (m *FlatHelper) Sub(prop string) *FlatHelper {
	return NewFlatHelper(m.Properties.MapOr(prop, nil))
}

// m := typed.New(e.Properties)

// 	if v, ok := m.StringIf("category"); ok {
// 		return []string{v}
// 	}

// 	// Slight modification of StringsIf so we capture
// 	// all string elements instead of blocking when there is none.
// 	// Tags can also be objects, such as tagged people as seen in
// 	// here: https://ownyourswarm.p3k.io/docs
// 	value, ok := m["category"]
// 	if !ok {
// 		return []string{}
// 	}

// 	if n, ok := value.([]string); ok {
// 		return n
// 	}

// 	if a, ok := value.([]interface{}); ok {
// 		n := []string{}
// 		for i := 0; i < len(a); i++ {
// 			if v, ok := a[i].(string); ok {
// 				n = append(n, v)
// 			}
// 		}
// 		return n
// 	}

// 	return []string{}

// func (j *JF2) GetOne(prop string) interface{} {
// 	// return
// }

// func (jf2 *JF2) AsStrings(prop string) []string {
// 	if val, ok := jf2.StringIf(prop); ok {
// 		return []string{val}
// 	}

// 	if vals, ok := jf2.StringsIf(prop); ok {
// 		return vals
// 	}

// 	return []string{}
// }

// func (jf2 *JF2) AsString(prop string) string {
// 	urls := jf2.AsStrings(prop)
// 	if len(urls) > 0 {
// 		return urls[0]
// 	}

// 	return ""
// }

// func (jf2 *JF2) Value(prop string) interface{} {
// 	return jf2.Typed[prop]
// }

// func (jf2 *JF2) Location() map[string]interface{} {
// 	// TODO: try checkin location, then location
// 	return nil
// }
