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
