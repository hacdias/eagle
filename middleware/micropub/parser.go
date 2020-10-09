package micropub

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/karlseguin/typed"
)

// https://github.com/hacdias/micropub-parser/blob/master/index.js

type Action string

const (
	ActionCreate   Action = "create" //nolint
	ActionUpdate          = "update"
	ActionDelete          = "delete"
	ActionUndelete        = "undelete"
)

type RequestUpdates struct {
	Replace map[string][]interface{}
	Add     map[string][]interface{}
	Delete  interface{}
}

type Request struct {
	Action     Action
	URL        string
	Type       string
	Properties typed.Typed
	Commands   typed.Typed
	Updates    *RequestUpdates
}

func parseFormEncodeed(body url.Values) (*Request, error) {
	req := &Request{
		Properties: map[string]interface{}{},
		Commands:   map[string]interface{}{},
	}

	if typ := body.Get("h"); typ != "" {
		req.Action = ActionCreate
		req.Type = "h-" + typ

		delete(body, "h")
		delete(body, "access_token")

		if _, ok := body["action"]; ok {
			return nil, errors.New("cannot specify an action when creating a post")
		}

		for key, val := range body {
			if len(val) == 0 {
				return nil, errors.New("values in form-encoded input can only be numeric indexed arrays")
			}

			if strings.HasPrefix(key, "mp-") {
				req.Commands[key] = val
			} else {
				req.Properties[key] = val
			}
		}

		return req, nil

	}

	if action := body.Get("action"); action != "" {
		if action == ActionUpdate {
			return nil, errors.New("micropub update actions require using the JSON syntax")
		}

		if url := body.Get("url"); url != "" {
			req.URL = url
		} else {
			return nil, errors.New("micropub actions require a URL property")
		}

		req.Action = Action(action)
		return req, nil
	}

	return nil, errors.New("no micropub data was found in the request")
}

type requestJSON struct {
	Type       []string                 `json:"type,omitempty"`
	URL        string                   `json:"url,omitempty"`
	Action     Action                   `json:"action,omitempty"`
	Properties map[string][]interface{} `json:"properties,omitempty"`
	Replace    map[string][]interface{} `json:"replace,omitempty"`
	Add        map[string][]interface{} `json:"add,omitempty"`
	Delete     interface{}              `json:"delete,omitempty"`
}

func parseJSON(body requestJSON) (*Request, error) {
	req := &Request{
		Properties: map[string]interface{}{},
		Commands:   map[string]interface{}{},
	}

	if body.Type != nil {
		if len(body.Type) != 1 {
			return nil, errors.New("type must have a single value")
		}

		req.Action = ActionCreate
		req.Type = body.Type[0]

		for key, value := range body.Properties {
			if len(value) == 0 {
				return nil, errors.New("property values in JSON format must be arrays")
			}

			if strings.HasPrefix(key, "mp-") {
				req.Commands[key] = value
			} else {
				req.Properties[key] = value
			}
		}

		return req, nil
	}

	if body.Action != "" {
		if body.URL == "" {
			return nil, errors.New("Micropub actions require a URL property")
		}

		req.Action = Action(body.Action)
		req.URL = body.URL

		if body.Action == ActionUpdate {
			req.Updates = &RequestUpdates{
				Add:     body.Add,
				Replace: body.Replace,
				Delete:  body.Delete,
			}
		}

		return req, nil
	}

	return nil, errors.New("no micropub data was found in the request")
}

func ParseRequest(r *http.Request) (*Request, error) {
	contentType := r.Header.Get("Content-type")
	if strings.Contains(contentType, "json") {
		req := requestJSON{}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			return nil, err
		}
		return parseJSON(req)
	}

	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	return parseFormEncodeed(r.Form)
}
