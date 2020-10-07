package services

import "github.com/hacdias/eagle/config"

type TweetOptions struct {
	ReplyTo    string
	Attachment string
}

type Twitter config.Twitter

func (t *Twitter) Like(id string) error {
	return nil
}

func (t *Twitter) Retweet(id string) error {
	return nil
}
func (t *Twitter) Tweet(status string, opts TweetOptions) error {
	return nil
}
