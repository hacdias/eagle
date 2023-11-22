package server

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/samber/lo"
	"go.hacdias.com/indielib/micropub"
)

// Flatten takes a Microformats map and flattens all arrays with
// a single value to one element.
func Flatten(data map[string][]any) map[string]any {
	return flatten(data).(map[string]any)
}

func flatten(data any) any {
	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		if value.Len() == 1 {
			return flatten(value.Index(0).Interface())
		}

		parsed := make([]any, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = flatten(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string]any{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			parsed[fmt.Sprint(k.Interface())] = flatten(v.Interface())
		}

		return parsed
	}

	return data
}

// Deflatten takes a flattened map and deflattens all single values to arrays.
func Deflatten(data map[string]any) map[string]any {
	return deflatten(data).(map[string]any)
}

func deflattenProperties(data any) map[string][]any {
	value := reflect.ValueOf(data)
	parsed := map[string][]any{}

	for _, k := range value.MapKeys() {
		v := value.MapIndex(k)
		key := fmt.Sprint(k.Interface())
		vk := reflect.TypeOf(v.Interface()).Kind()

		if vk == reflect.Slice || vk == reflect.Array {
			parsed[key] = deflatten(v.Interface()).([]any)
		} else {
			parsed[key] = []any{deflatten(v.Interface())}
		}
	}

	return parsed
}

func deflatten(data any) any {
	if data == nil {
		return []any{nil}
	}

	value := reflect.ValueOf(data)
	kind := value.Kind()

	if kind == reflect.Slice {
		parsed := make([]any, value.Len())

		for i := 0; i < value.Len(); i++ {
			parsed[i] = deflatten(value.Index(i).Interface())
		}

		return parsed
	}

	if kind == reflect.Map {
		parsed := map[string]any{}

		for _, k := range value.MapKeys() {
			v := value.MapIndex(k)
			key := fmt.Sprint(k.Interface())
			vk := reflect.TypeOf(v.Interface()).Kind()

			if key == "properties" {
				parsed[key] = deflattenProperties(v.Interface())
			} else if key == "value" || key == "html" {
				parsed[key] = deflatten(v.Interface())
			} else if vk == reflect.Slice || vk == reflect.Array {
				parsed[key] = deflatten(v.Interface()).([]any)
			} else {
				parsed[key] = []any{deflatten(v.Interface())}
			}
		}

		return parsed
	}

	return data
}

// Update updates a set of existing properties with the new request.
func Update(properties map[string][]any, req micropub.RequestUpdate) (map[string][]any, error) {
	if req.Replace != nil {
		for key, value := range req.Replace {
			properties[key] = value
		}
	}

	if req.Add != nil {
		for key, value := range req.Add {
			switch key {
			case "name":
				return nil, errors.New("cannot add a new name")
			case "content":
				return nil, errors.New("cannot add content")
			default:
				if key == "published" {
					if _, ok := properties["published"]; ok {
						return nil, errors.New("cannot replace published through add method")
					}
				}

				if _, ok := properties[key]; !ok {
					properties[key] = []any{}
				}

				properties[key] = append(properties[key], value...)
			}
		}
	}

	if req.Delete != nil {
		if reflect.TypeOf(req.Delete).Kind() == reflect.Slice {
			toDelete, ok := req.Delete.([]any)
			if !ok {
				return nil, errors.New("invalid delete array")
			}

			for _, key := range toDelete {
				delete(properties, fmt.Sprint(key))
			}
		} else {
			toDelete, ok := req.Delete.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid delete object: expected map[string]any, got: %s", reflect.TypeOf(req.Delete))
			}

			for key, v := range toDelete {
				value, ok := v.([]any)
				if !ok {
					return nil, fmt.Errorf("invalid value: expected []any, got: %s", reflect.TypeOf(value))
				}

				if _, ok := properties[key]; !ok {
					properties[key] = []any{}
				}

				properties[key] = lo.Filter(properties[key], func(ss any, _ int) bool {
					for _, s := range value {
						if s == ss {
							return false
						}
					}
					return true
				})
			}
		}
	}

	return properties, nil
}
