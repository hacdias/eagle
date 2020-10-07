package services

import (
	"net/http"
	"time"

	"github.com/hacdias/eagle/config"
)

type WebmentionPayload struct {
	Secret  string `json:"secret"`
	Source  string `json:"source"`
	Deleted bool   `json:"deleted"`
	Target  string `json:"target"`
	Post    struct {
		Type   string `json:"type"`
		Author struct {
			Name  string `json:"name"`
			Photo string `json:"photo"`
			URL   string `json:"url"`
		} `json:"author"`
		URL        string    `json:"url"`
		Published  time.Time `json:"published"`
		Name       string    `json:"name"`
		RepostOf   string    `json:"repost-of"`
		WmProperty string    `json:"wm-property"`
	} `json:"post"`
}

type Webmentions struct {
	Domain    string
	Telegraph config.Telegraph
	Git       Git
	Media     Media
	Hugo      Hugo
}

func (w *Webmentions) Send(source string, targets ...string) error {
	for _, target := range targets {
		req, err := http.NewRequest(http.MethodPost, "https://telegraph.p3k.io/webmention", nil)
		if err != nil {
			// TODO: log
			continue
		}

		req.PostForm.Add("token", w.Telegraph.Token)
		req.PostForm.Add("source", source)
		req.PostForm.Add("target", target)
		req.Header.Set("Accept", "application/json")

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			// TODO: log
		} else {
			// TODO: log
		}
	}

	return nil
}

func (w *Webmentions) Receive(payload *WebmentionPayload) error {
	return nil
}
