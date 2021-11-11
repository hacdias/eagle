package micropub

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/thoas/go-funk"
)

// Update updates a set of existing properties with the new request.
func Update(properties map[string][]interface{}, req *Request) (map[string][]interface{}, error) {
	if req.Updates.Replace != nil {
		for key, value := range req.Updates.Replace {
			properties[key] = value
		}
	}

	if req.Updates.Add != nil {
		for key, value := range req.Updates.Add {
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
					properties[key] = []interface{}{}
				}

				properties[key] = append(properties[key], value...)
			}
		}
	}

	if req.Updates.Delete != nil {
		if reflect.TypeOf(req.Updates.Delete).Kind() == reflect.Slice {
			toDelete, ok := req.Updates.Delete.([]interface{})
			if !ok {
				return nil, errors.New("invalid delete array")
			}

			for _, key := range toDelete {
				delete(properties, fmt.Sprint(key))
			}
		} else {
			toDelete, ok := req.Updates.Delete.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid delete object: expected map[string]interface{}, got: %s", reflect.TypeOf(req.Updates.Delete))
			}

			for key, v := range toDelete {
				value, ok := v.([]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid value: expected []interface{}, got: %s", reflect.TypeOf(value))
				}

				if _, ok := properties[key]; !ok {
					properties[key] = []interface{}{}
				}

				properties[key] = funk.Filter(properties[key], func(ss interface{}) bool {
					for _, s := range value {
						if s == ss {
							return false
						}
					}
					return true
				}).([]interface{})
			}
		}
	}

	return properties, nil
}
