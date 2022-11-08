package mf2

import (
	"fmt"
	"reflect"
)

// Flatten takes a Microformats map and flattens all arrays with
// a single value to one element.
func Flatten(data map[string][]interface{}) map[string]interface{} {
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
			v := value.MapIndex(k)
			parsed[fmt.Sprint(k.Interface())] = flatten(v.Interface())
		}

		return parsed
	}

	return data
}

// Deflatten takes a flattened map and deflattens all single values to arrays.
func Deflatten(data map[string]interface{}) map[string]interface{} {
	return deflatten(data).(map[string]interface{})
}

func deflattenProperties(data interface{}) map[string][]interface{} {
	value := reflect.ValueOf(data)
	parsed := map[string][]interface{}{}

	for _, k := range value.MapKeys() {
		v := value.MapIndex(k)
		key := fmt.Sprint(k.Interface())
		vk := reflect.TypeOf(v.Interface()).Kind()

		if vk == reflect.Slice || vk == reflect.Array {
			parsed[key] = deflatten(v.Interface()).([]interface{})
		} else {
			parsed[key] = []interface{}{deflatten(v.Interface())}
		}
	}

	return parsed
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
		parsed := map[string]interface{}{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			key := fmt.Sprint(k.Interface())
			vk := reflect.TypeOf(v.Interface()).Kind()

			if key == "properties" {
				parsed[key] = deflattenProperties(v.Interface())
			} else if key == "value" || key == "html" {
				parsed[key] = deflatten(v.Interface())
			} else if vk == reflect.Slice || vk == reflect.Array {
				parsed[key] = deflatten(v.Interface()).([]interface{})
			} else {
				parsed[key] = []interface{}{deflatten(v.Interface())}
			}
		}

		return parsed
	}

	return data
}
