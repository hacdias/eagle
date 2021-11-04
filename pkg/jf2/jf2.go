package jf2

import "github.com/karlseguin/typed"

type JF2 struct {
	typed.Typed
	typ  Type
	prop string
}

func NewJF2(data map[string]interface{}) *JF2 {
	return &JF2{
		Typed: typed.New(data),
	}
}

func (jf2 *JF2) Type() Type {
	if jf2.typ == "" {
		jf2.typ, jf2.prop = DiscoverType(jf2.Typed)
	}

	return jf2.typ
}

func (jf2 *JF2) Property() string {
	if jf2.prop == "" {
		jf2.typ, jf2.prop = DiscoverType(jf2.Typed)
	}

	return jf2.prop
}

func (jf2 *JF2) AsStrings(prop string) []string {
	if val, ok := jf2.StringIf(prop); ok {
		return []string{val}
	}

	if vals, ok := jf2.StringsIf(prop); ok {
		return vals
	}

	return []string{}
}

func (jf2 *JF2) AsString(prop string) string {
	urls := jf2.AsStrings(prop)
	if len(urls) > 0 {
		return urls[0]
	}

	return ""
}

func (jf2 *JF2) Location() map[string]interface{} {
	// TODO: try checkin location, then location
	return nil
}
