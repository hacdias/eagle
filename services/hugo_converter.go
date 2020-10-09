package services

import (
	"reflect"
)

func (h *Hugo) mf2ToInternal(data interface{}) interface{} {
	kind := reflect.ValueOf(data).Kind()

	if kind == reflect.Array || kind == reflect.Slice {
		v := data.([]interface{})
		if len(v) == 1 {
			return h.mf2ToInternal(v[0])
		}

		newData := make([]interface{}, len(v))
		for i, v := range v {
			newData[i] = h.mf2ToInternal(v)
		}
		return newData
	}

	if kind == reflect.Map {
		parsed := map[string]interface{}{}

		for key, value := range data.(map[string]interface{}) {
			parsed[key] = h.mf2ToInternal(value)
		}

		return parsed
	}

	return data

}

func (h *Hugo) internalToMf2(data interface{}) interface{} {
	if data == nil {
		return []interface{}{nil}
	}

	kind := reflect.ValueOf(data).Kind()
	if kind == reflect.Array || kind == reflect.Slice {
		v := data.([]interface{})
		newData := make([]interface{}, len(v))
		for i, v := range v {
			newData[i] = h.internalToMf2(v)
		}
		return newData
	}

	if kind == reflect.Map {
		parsed := map[string][]interface{}{}

		for key, value := range data.(map[string]interface{}) {
			kind := reflect.ValueOf(value).Kind()

			if kind == reflect.Array || kind == reflect.Slice || key == "properties" || key == "value" {
				parsed[key] = h.internalToMf2(value).([]interface{})
			} else {
				parsed[key] = []interface{}{h.internalToMf2(value)}
			}

		}

		return parsed
	}

	return data.(map[string][]interface{})
}
