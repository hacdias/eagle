package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hacdias/eagle"
)

const activityDateLayout = "2006-01-02T15:04:05-07:00"
const activityContentType = "application/activity+json"
const activityExt = ".as2"

type activity struct {
	Context           []string   `json:"@context,omitempty"`
	ID                string     `json:"id,omitempty"`
	URL               string     `json:"url,omitempty"`
	Type              string     `json:"type,omitempty"`
	Name              string     `json:"name,omitempty"`
	Summary           string     `json:"summary,omitempty"`
	PreferredUsername string     `json:"preferredUsername,omitempty"`
	Inbox             string     `json:"inbox,omitempty"`
	Outbox            string     `json:"outbox,omitempty"`
	Icon              *activity  `json:"icon,omitempty"`
	Image             *activity  `json:"image,omitempty"`
	Href              string     `json:"href,omitempty"`
	MediaType         string     `json:"mediaType,omitempty"`
	Owner             string     `json:"Owner,omitempty"`
	PublicKeyPem      string     `json:"publicKeyPem,omitempty"`
	To                []string   `json:"to,omitempty"`
	Published         string     `json:"published,omitempty"`
	Updated           string     `json:"updated,omitempty"`
	Content           string     `json:"content,omitempty"`
	AttributedTo      string     `json:"attributedTo,omitempty"`
	InReplyTo         string     `json:"inReplyTo,omitempty"`
	Tags              []activity `json:"tags,omitempty"`
	Attachment        []activity `json:"attachment,omitempty"`
}

func (s *Server) serveEntryActivity(w http.ResponseWriter, entry *eagle.Entry) {
	data := &activity{
		Context: []string{
			"https://www.w3.org/ns/activitystreams",
		},
		To: []string{
			"https://www.w3.org/ns/activitystreams#Public",
		},
		ID:           entry.Permalink,
		URL:          entry.Permalink,
		MediaType:    "text/html",
		AttributedTo: s.c.Site.Domain,
		Tags:         []activity{},
	}

	if entry.Metadata.Title != "" {
		data.Name = entry.Metadata.Title
	}

	if !entry.Metadata.PublishDate.IsZero() {
		data.Published = entry.Metadata.PublishDate.Format(activityDateLayout)
	}

	if !entry.Metadata.UpdateDate.IsZero() {
		data.Updated = entry.Metadata.UpdateDate.Format(activityDateLayout)
	}

	if entry.Section == "articles" {
		data.Type = "Article"
	} else {
		data.Type = "Note"
	}

	if entry.Metadata.ReplyTo != "" {
		data.InReplyTo = entry.Metadata.ReplyTo
	}

	for _, tag := range entry.Metadata.Tags {
		data.Tags = append(data.Tags, activity{
			Type: "Hashtag",
			ID:   fmt.Sprintf("%s/tags/%s", s.Site.Domain, tag), // TODO: check correctness to build URLs
		})
	}

	/*
		TODO for entry.Mentions ?
		{{ $res = $res | append (dict "type" "Mention" "href" $mention.href "name" $mention.name) }}
	*/

	/*

	  "content": {{ partialCached "cleaned-content.html" . .Permalink | jsonify }},
	}*/

	/*
		TODO: attachments
		{{- with .Params.pictures }}
		{{- range . }}
		  {{ $url := printf "https://cdn.hacdias.com/photos/t/%s-2000x.jpeg" .slug }}
		  {{ $attachments = $attachments | append (dict "mediaType" "image/jpeg" "type" "Image" "url" $url "name" .title ) }}
		{{- end }}
		{{- end }}
	*/

	w.Header().Set("Content-Type", activityContentType)
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.Errorf("error while serving json: %s", err)
	}
}

func (s *Server) serveHomeActivity(w http.ResponseWriter) {
	data := &activity{
		Context: []string{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		ID:                s.c.Site.Domain,
		URL:               s.c.Site.Domain,
		Type:              "Person",
		Name:              s.c.Site.Author.Name,
		Summary:           s.c.Site.Description,
		PreferredUsername: s.Site.Author.Username,
	}

	if s.c.Site.Author.Avatar != "" {
		data.Icon = &activity{
			Type: "image",
			URL:  s.c.Site.Author.Avatar,
		}
	}

	if s.c.Site.Author.Cover != "" {
		data.Image = &activity{
			Type:      "image",
			MediaType: "image/jpeg",
			URL:       s.c.Site.Author.Cover,
		}
	}

	/*
	  {{- with .Site.Params.activityPub.inbox -}},
	  "inbox": "{{ . }}"
	  {{- end }}
	  {{- with .Site.Params.activityPub.outbox -}},
	  "outbox": "{{ . }}"
	  {{- end }}
	  {{- with .Site.Params.activityPub.publicKeyPem -}},
	  "publicKey": {
	      "id": "{{ "" | absLangURL }}#key",
	      "owner": "{{ "" | absLangURL }}",
	      "publicKeyPem": "{{ . }}"
	  }
	  {{- end }}
	}*/

	w.Header().Set("Content-Type", activityContentType)
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.Errorf("error while serving json: %s", err)
	}
}
