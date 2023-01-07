package mf2

import (
	"reflect"

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

	if _, ok := data["properties"].(map[string]interface{}); !ok {
		data = map[string]interface{}{
			"properties": data,
		}
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

func (m *FlatHelper) Strings(prop string) []string {
	if v, ok := m.Properties.StringIf(prop); ok {
		return []string{v}
	}

	if v, ok := m.Properties.StringsIf(prop); ok && len(v) > 0 {
		return v
	}

	return []string{}
}

func (m *FlatHelper) Float(prop string) float64 {
	if v, ok := m.Properties.FloatIf(prop); ok {
		return v
	}

	if v, ok := m.Properties.FloatsIf(prop); ok && len(v) > 0 {
		return v[0]
	}

	return 0
}

func (m *FlatHelper) Int(prop string) int {
	if v, ok := m.Properties.IntIf(prop); ok {
		return v
	}

	if v, ok := m.Properties.IntsIf(prop); ok && len(v) > 0 {
		return v[0]
	}

	return 0
}

func (m *FlatHelper) media(prop string) []map[string]interface{} {
	v, ok := m.Properties[prop]
	if !ok {
		return nil
	}

	if vv, ok := v.(string); ok {
		return []map[string]interface{}{
			{
				"value": vv,
				"alt":   "",
			},
		}
	}

	value := reflect.ValueOf(v)
	kind := value.Kind()
	parsed := []map[string]interface{}{}

	if kind == reflect.Array || kind == reflect.Slice {
		for i := 0; i < value.Len(); i++ {
			v = value.Index(i).Interface()

			if vv, ok := v.(string); ok {
				parsed = append(parsed, map[string]interface{}{
					"value": vv,
					"alt":   "",
				})
			} else if vv, ok := v.(map[string]interface{}); ok {
				parsed = append(parsed, vv)
			}
		}
	}

	return parsed
}

func (m *FlatHelper) Audios() []map[string]interface{} {
	return m.media("audio")
}

func (m *FlatHelper) Photos() []map[string]interface{} {
	return m.media("photo")
}

func (m *FlatHelper) Videos() []map[string]interface{} {
	return m.media("video")
}

func (m *FlatHelper) Photo() map[string]interface{} {
	photos := m.Photos()
	if len(photos) > 0 {
		return photos[0]
	}

	return nil
}

func (m *FlatHelper) Name() string {
	return m.Properties.StringOr("name", "")
}

func (m *FlatHelper) Sub(prop string) *FlatHelper {
	return NewFlatHelper(m.Properties.MapOr(prop, nil))
}

func (m *FlatHelper) Subs(prop string) []*FlatHelper {
	if v := m.Properties.Map(prop); v != nil {
		return []*FlatHelper{NewFlatHelper(v)}
	}

	vv := m.Properties.Maps(prop)
	var hh []*FlatHelper
	for _, v := range vv {
		hh = append(hh, NewFlatHelper(v))
	}

	return hh
}
