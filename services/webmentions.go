package services

import (
	"net/http"

	"github.com/hacdias/eagle/config"
)

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

func (w *Webmentions) Receive() {

}
