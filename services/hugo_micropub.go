package services

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/hacdias/eagle/middleware/micropub"
)

var typesWithLinks = map[micropub.Type]string{
	micropub.TypeRepost:   "repost-of",
	micropub.TypeLike:     "like-of",
	micropub.TypeReply:    "in-reply-to",
	micropub.TypeBookmark: "bookmark-of",
}

func (h *Hugo) FromMicropub(post *micropub.Request) (*HugoEntry, *Syndication, error) {
	entry := &HugoEntry{
		Content:  "",
		Metadata: map[string]interface{}{},
	}

	if published, ok := post.Properties.StringIf("published"); ok {
		entry.Metadata["date"] = published
	} else {
		entry.Metadata["date"] = time.Now().Format(time.RFC3339)
	}

	postType := micropub.DiscoverType(post.Properties)

	switch postType {
	case micropub.TypeReply, micropub.TypeNote, micropub.TypeArticle:
		// It's fine.
	default:
		return nil, nil, errors.New("type not supported " + string(postType))
	}

	if content, ok := post.Properties.StringsIf("content"); ok {
		// TODO: check content like { html: , text: }. Return unsupported for that.
		entry.Content = strings.TrimSpace(strings.Join(content, "\n"))
	}

	if name, ok := post.Properties.StringsIf("name"); ok {
		entry.Metadata["title"] = strings.TrimSpace(strings.Join(name, " "))
	}

	delete(post.Properties, "published")
	delete(post.Properties, "content")
	delete(post.Properties, "name")

	synd := &Syndication{
		Type:    postType,
		Related: []string{},
		Targets: []string{},
	}

	switch postType {
	case micropub.TypeRepost, micropub.TypeLike, micropub.TypeReply, micropub.TypeBookmark:
		links, ok := post.Properties.StringsIf(typesWithLinks[postType])
		if !ok {
			return nil, nil, errors.New("type " + string(postType) + " must refer to some link")
		}
		var err error
		synd.Related, err = cleanRelated(links)
		if err != nil {
			return nil, nil, err
		}

		if len(synd.Related) > 0 {
			post.Properties[typesWithLinks[postType]] = synd.Related
		}
	}

	if targets, ok := post.Commands.StringsIf("mp-syndicate-to"); ok {
		synd.Targets = targets
	}

	if categories, ok := post.Properties.StringsIf("category"); ok {
		entry.Metadata["tags"] = categories
		delete(post.Properties, "category")
	}

	entry.Metadata["properties"] = post.Properties

	if slugSlice, ok := post.Commands.StringsIf("mp-slug"); ok && len(slugSlice) == 1 {
		slug := strings.TrimSpace(strings.Join(slugSlice, "\n"))

		section := "micro"
		switch postType {
		case micropub.TypeArticle:
			section = "article"
		}

		year := time.Now().Year()
		month := time.Now().Month()
		entry.ID = fmt.Sprintf("/%s/%04d/%02d/%s", section, year, month, slug)
		permalink, err := h.makeURL(entry.ID)
		if err != nil {
			return nil, nil, err
		}
		entry.Permalink = permalink
	} else {
		return nil, nil, errors.New("post must have a slug")
	}

	return entry, synd, nil
}

func cleanRelated(urls []string) ([]string, error) {
	clean := make([]string, len(urls))

	for i, u := range urls {
		if strings.HasPrefix(u, "https://twitter.com") && strings.Contains(u, "/status/") {
			u, err := url.Parse(u)
			if err != nil {
				return nil, err
			}

			for k := range u.Query() {
				u.Query().Del(k)
			}

			clean[i] = u.String()
		} else {
			clean[i] = u
		}
	}

	return clean, nil
}

func interfacesToStrings(data []interface{}) []string {
	res := []string{}

	for _, v := range data {
		switch v := v.(type) {
		case string:
			res = append(res, v)
		default:
			res = append(res, fmt.Sprint(v))
		}
	}

	return res
}

func (e *HugoEntry) Update(mr *micropub.Request) error {
	tags := []string{}
	props := map[string][]interface{}{}

	if t, ok := e.Metadata.StringsIf("tags"); ok {
		tags = t
	}

	if p, ok := e.Metadata.InterfaceIf("properties"); ok {
		props, ok = p.(map[string][]interface{})
		if !ok {
			return errors.New("invalid properties on entry")
		}
	}

	if mr.Updates.Replace != nil {
		for key, value := range mr.Updates.Replace {
			switch key {
			case "name":
				strs := interfacesToStrings(value)
				e.Metadata["title"] = strings.TrimSpace(strings.Join(strs, " "))
			case "category":
				tags = interfacesToStrings(value)
			case "content":
				strs := interfacesToStrings(value)
				e.Content = strings.TrimSpace(strings.Join(strs, " "))
			case "published":
				_, hasDate := e.Metadata["date"]
				_, hasPublishDate := e.Metadata["publishDate"]

				if !hasPublishDate && hasDate {
					e.Metadata["publishDate"] = e.Metadata["date"]
				}

				strs := interfacesToStrings(value)
				e.Metadata["date"] = strings.TrimSpace(strings.Join(strs, " "))
			default:
				props[key] = value
			}
		}
	}

	if mr.Updates.Add != nil {
		for key, value := range mr.Updates.Add {
			switch key {
			case "name":
				return errors.New("cannot add a new name")
			case "category":
				tags = append(tags, interfacesToStrings(value)...)
			case "content":
				strs := interfacesToStrings(value)
				e.Content += strings.TrimSpace(strings.Join(strs, " "))
			case "published":
				if _, ok := e.Metadata["date"]; ok {
					return errors.New("cannot replace published through add method")
				}
				strs := interfacesToStrings(value)
				e.Metadata["date"] = strings.TrimSpace(strings.Join(strs, " "))
			default:
				if _, ok := props[key]; !ok {
					props[key] = []interface{}{}
				}

				props[key] = append(props[key], value...)
			}
		}
	}

	if mr.Updates.Delete != nil {
		if reflect.TypeOf(mr.Updates.Delete).Kind() == reflect.Slice {
			toDelete, ok := mr.Updates.Delete.([]interface{})
			if !ok {
				return errors.New("invalid delete array")
			}

			for _, key := range toDelete {
				switch key {
				case "category":
					tags = []string{}
				case "content":
					e.Content = ""
				default:
					delete(props, fmt.Sprint(key))
				}
			}
		} else {
			toDelete, ok := mr.Updates.Delete.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid delete object: expected map[string]interaface{}, got: %s", reflect.TypeOf(mr.Updates.Delete))
			}

			for key, v := range toDelete {
				value, ok := v.([]interface{})
				if !ok {
					return fmt.Errorf("invalid value: expected []interaface{}, got: %s", reflect.TypeOf(value))
				}

				switch key {
				case "content":
					e.Content = ""
				case "category":
					tags = filter(tags, func(ss interface{}) bool {
						for _, s := range value {
							if s == ss {
								return false
							}
						}
						return true
					}).([]string)
				default:
					if _, ok := props[key]; !ok {
						props[key] = []interface{}{}
					}

					props[key] = filter(props[key], func(ss interface{}) bool {
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
	}

	e.Metadata["tags"] = tags
	e.Metadata["properties"] = props
	return nil
}

func filter(arr interface{}, cond func(interface{}) bool) interface{} {
	contentType := reflect.TypeOf(arr)
	contentValue := reflect.ValueOf(arr)

	newContent := reflect.MakeSlice(contentType, 0, 0)
	for i := 0; i < contentValue.Len(); i++ {
		if content := contentValue.Index(i); cond(content.Interface()) {
			newContent = reflect.Append(newContent, content)
		}
	}
	return newContent.Interface()
}
