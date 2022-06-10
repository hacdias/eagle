package syndicator

import (
	"context"
	"errors"
	"fmt"
	urlpkg "net/url"
	"strings"

	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/entry/mf2"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type Reddit struct {
	User   string
	client *reddit.Client
}

func NewReddit(client *reddit.Client) *Reddit {
	reddit := &Reddit{
		User:   client.Username,
		client: client,
	}

	return reddit
}

func (r *Reddit) Syndicate(entry *entry.Entry) (url string, err error) {
	if r.isSyndicated(entry) {
		// If it is already syndicated to Reddit, do not try to syndicate again.
		return "", errors.New("cannot re-syndicate to Reddit")
	}

	mm := entry.Helper()
	typ := mm.PostType()

	if typ == mf2.TypeLike {
		return r.upvote(entry)
	}

	if typ == mf2.TypeReply {
		return r.reply(entry)
	}

	return r.post(entry)
}

func (r *Reddit) IsByContext(entry *entry.Entry) bool {
	if r.isSyndicated(entry) {
		// If it is already syndicated to Reddit, do not try to syndicate again.
		return false
	}

	mm := entry.Helper()
	typ := mm.PostType()

	switch typ {
	case mf2.TypeReply, mf2.TypeLike:
	default:
		return false
	}

	urlStr := mm.String(mm.TypeProperty())
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return false
	}

	return strings.Contains(url.Host, "reddit.com")
}

func (r *Reddit) Name() string {
	return fmt.Sprintf("Reddit (%s)", r.User)
}

func (r *Reddit) Identifier() string {
	return fmt.Sprintf("reddit-%s", r.User)
}

func (r *Reddit) isSyndicated(entry *entry.Entry) bool {
	mm := entry.Helper()

	syndications := mm.Strings("syndication")
	for _, syndication := range syndications {
		url, _ := urlpkg.Parse(syndication)
		if url != nil && strings.Contains(url.Host, "reddit.com") {
			return true
		}
	}

	return false
}

func (r *Reddit) idFromUrl(urlStr string) (string, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return "", err
	}

	path := strings.TrimSuffix(url.Path, "/")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 2 {
		// Subreddit
		return parts[1], nil
	} else if len(parts) == 5 {
		// Post
		return "t3_" + parts[3], nil
	} else if len(parts) == 6 {
		// Comment
		return "t1_" + parts[5], nil
	}

	return "", errors.New("could not get id from Reddit URL")
}

func (r *Reddit) upvote(entry *entry.Entry) (string, error) {
	mm := entry.Helper()
	urlStr := mm.String(mm.TypeProperty())
	id, err := r.idFromUrl(urlStr)
	if err != nil {
		return "", err
	}

	_, err = r.client.Post.Upvote(context.Background(), id)
	if err != nil {
		return "", err
	}

	return urlStr, nil
}

func (r *Reddit) reply(entry *entry.Entry) (string, error) {
	mm := entry.Helper()
	urlStr := mm.String(mm.TypeProperty())
	id, err := r.idFromUrl(urlStr)
	if err != nil {
		return "", err
	}

	comment, _, err := r.client.Comment.Submit(context.Background(), id, entry.Content)
	if err != nil {
		return "", err
	}

	return "https://www.reddit.com" + comment.Permalink, nil
}

func (r *Reddit) post(entry *entry.Entry) (string, error) {
	audience := entry.Audience()
	if len(audience) != 1 {
		return "", errors.New("audience needs to have exactly one element for reddit syndication")
	}

	subreddit, err := r.idFromUrl(audience[0])
	if err != nil {
		return "", err
	}

	post, _, err := r.client.Post.SubmitText(context.Background(), reddit.SubmitTextRequest{
		Subreddit: subreddit,
		Title:     entry.Title,
		Text:      entry.Content,
	})
	if err != nil {
		return "", err
	}

	return post.URL, nil
}
