package mf2

import "github.com/karlseguin/typed"

type JF2 struct {
	Map          typed.Typed
	PostType     Type
	TypeProperty string
}

func NewJF2(data map[string]interface{}) *JF2 {
	typed := typed.New(data)

	// This should receive the entire thing I guess.
	typ, prop := DiscoverType(typed.MapOr("properties", map[string]interface{}{}))

	return &JF2{
		Map:          typed,
		PostType:     typ,
		TypeProperty: prop,
	}
}

// func (j *JF2) Type() string {
// 	return j.Map.StringOr("type", "")
// }

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
