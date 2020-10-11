package services

import (
	"fmt"
	"reflect"
)

func mf2ToInternal(data interface{}) interface{} {
	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		if value.Len() == 1 {
			return mf2ToInternal(value.Index(0).Interface())
		}

		parsed := make([]interface{}, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = mf2ToInternal(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string]interface{}{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			parsed[fmt.Sprint(k.Interface())] = mf2ToInternal(v.Interface())
		}

		return parsed
	}

	return data
}

func internalToMf2(data interface{}) interface{} {
	if data == nil {
		return []interface{}{nil}
	}

	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		parsed := make([]interface{}, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = internalToMf2(value.Index(i).Interface())
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
				parsed[key] = internalToMf2(v.Interface()).([]interface{})
			} else {
				parsed[key] = []interface{}{internalToMf2(v.Interface())}
			}
		}

		return parsed
	}

	return data
}
