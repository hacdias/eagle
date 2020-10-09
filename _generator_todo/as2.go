package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"reflect"

	"willnorris.com/go/microformats"
)

type as2 struct {
	Context      []string `json:"@context,omitempty"`
	To           []string `json:"to,omitempty"`
	Published    string   `json:"published,omitempty"`
	Updated      string   `json:"updated,omitempty"`
	ID           string   `json:"id,omitempty"`
	URL          string   `json:"url,omitempty"`
	Content      string   `json:"content,omitempty"`
	MediaType    string   `json:"mediaType,omitempty"`
	Name         string   `json:"name,omitempty"`
	Type         string   `json:"type,omitempty"`
	AttributedTo string   `json:"attributedTo,omitempty"`
	InReplyTo    string   `json:"inReplyTo,omitempty"`
}

func generateAs2(data *microformats.Data, url, base string) error {
	if len(data.Items) != 1 {
		log.Printf("invalid number of mf2 items for %s", base)
		return nil
	}

	a := &as2{
		Context: []string{
			"https://www.w3.org/ns/activitystreams",
		},
		To: []string{
			"https://www.w3.org/ns/activitystreams#Public",
		},
		MediaType: "text/html",
		ID:        url,
		URL:       url,
	}

	content := data.Items[0].Properties["content"]
	if len(content) != 1 {
		log.Printf("Invalid content size for %s", base)
		return nil
	}

	switch content[0].(type) {
	case map[string]string:
		a.Content = content[0].(map[string]string)["html"]
		if a.Content == "" {
			log.Println("invalid content")
		}
	default:

		fmt.Println(content[0])
		fmt.Println(reflect.TypeOf(content[0]).Name())
	}

	aa, err := json.Marshal(a)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(base, "index.as2"), aa, 0644)
}

/*

/*
{
  "published": {{ dateFormat "2006-01-02T15:04:05-07:00" .Date | jsonify }},
  "updated": {{ dateFormat "2006-01-02T15:04:05-07:00" .Lastmod | jsonify }},
  {{ with .Title }}"name": {{ . | jsonify }},{{ end }}
  "type": "{{ if eq .Section "articles" }}Article{{ else }}Note{{ end }}",
  "attributedTo": "{{ "" | absLangURL }}"{{ if .Params.properties }}{{ if isset .Params.properties "in-reply-to" }},
  "inReplyTo": "{{ index .Params.properties "in-reply-to" }}"
  {{ end }}{{ end }}
}
*/
