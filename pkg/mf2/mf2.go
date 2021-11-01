package mf2

import (
	"fmt"
	"reflect"
)

// Flatten takes a interface{} Microformats2 and flattens all
// arrays that only have one element.
func Flatten(data interface{}) interface{} {
	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		if value.Len() == 1 {
			return Flatten(value.Index(0).Interface())
		}

		parsed := make([]interface{}, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = Flatten(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string]interface{}{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			parsed[fmt.Sprint(k.Interface())] = Flatten(v.Interface())
		}

		return parsed
	}

	return data
}

// Deflatten takes an interface{} and deflattens all single values
// to arrays (except for keys such as "value" and "properties") to ensure
// that the provided data is compatible with Microformats2.
func Deflatten(data interface{}) interface{} {
	if data == nil {
		return []interface{}{nil}
	}

	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		parsed := make([]interface{}, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = Deflatten(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string]interface{}{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			key := fmt.Sprint(k.Interface())
			vk := reflect.TypeOf(v.Interface()).Kind()

			if key == "properties" || key == "value" || vk == reflect.Slice || vk == reflect.Array {
				parsed[key] = Deflatten(v.Interface()).([]interface{})
			} else {
				parsed[key] = []interface{}{Deflatten(v.Interface())}
			}
		}

		return parsed
	}

	return data
}
