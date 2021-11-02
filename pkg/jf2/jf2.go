// Package jf2 provides functionality to convert between Multiformats2 and the JF2
// simplified format.
// 	- https://jf2.spec.indieweb.org/
// 	- https://indieweb.org/jf2
package jf2

import (
	"fmt"
	"reflect"
	"strings"
)

// FromMicroformats takes a Microformats map and flattens all arrays with
// a single value to one element.
//
// TODO: handle reserved properties and ensure their format:
// - https://jf2.spec.indieweb.org/#reservedproperties
func FromMicroformats(data map[string][]interface{}) map[string]interface{} {
	return flatten(data).(map[string]interface{})
}

func flatten(data interface{}) interface{} {
	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		if value.Len() == 1 {
			return flatten(value.Index(0).Interface())
		}

		parsed := make([]interface{}, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = flatten(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string]interface{}{}

		for _, k := range value.MapKeys() {
			key := fmt.Sprint(k.Interface())
			val := flatten(value.MapIndex(k).Interface())

			if key == "type" {
				if t, ok := val.(string); ok {
					val = strings.TrimPrefix(t, "h-")
				}
			}

			parsed[key] = val
		}

		return parsed
	}

	return data
}

// ToMicroformats takes a JF2 map and deflattens all single values to arrays.
func ToMicroformats(data map[string]interface{}) map[string][]interface{} {
	return deflatten(data).(map[string][]interface{})
}

func deflatten(data interface{}) interface{} {
	if data == nil {
		return []interface{}{nil}
	}

	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		parsed := make([]interface{}, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = deflatten(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string][]interface{}{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			key := fmt.Sprint(k.Interface())
			vk := reflect.TypeOf(v.Interface()).Kind()

			if key == "properties" || key == "value" || vk == reflect.Slice || vk == reflect.Array {
				parsed[key] = deflatten(v.Interface()).([]interface{})
			} else {
				parsed[key] = []interface{}{deflatten(v.Interface())}
			}
		}

		return parsed
	}

	return data
}
