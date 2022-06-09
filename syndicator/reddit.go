package syndicator

import (
	"context"
	"errors"
	"fmt"
	"html"
	urlpkg "net/url"
	"strings"
	"time"

	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/entry"
	"github.com/hacdias/eagle/v3/entry/mf2"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

type Reddit struct {
	User   string
	client *reddit.Client
}

func NewReddit(opts *config.Reddit) (*Reddit, error) {
	credentials := reddit.Credentials{
		ID:       opts.App,
		Secret:   opts.Secret,
		Username: opts.User,
		Password: opts.Password,
	}

	client, err := reddit.NewClient(credentials)
	if err != nil {
		return nil, err
	}

	reddit := &Reddit{
		User:   opts.User,
		client: client,
	}

	return reddit, nil
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

func (r *Reddit) GetXRay(urlStr string) (map[string]interface{}, error) {
	id, err := r.idFromUrl(urlStr)
	if err != nil {
		return nil, err
	}

	posts, comments, _, _, err := r.client.Listings.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(id, "t1_") {
		if len(comments) != 1 {
			return nil, errors.New("comment not found")
		}

		content := html.UnescapeString(comments[0].Body)
		if content == "[deleted]" {
			return nil, errors.New("comment was deleted")
		}

		data := map[string]interface{}{
			"content":   content,
			"published": comments[0].Created.Time.Format(time.RFC3339),
			"url":       "https://www.reddit.com" + comments[0].Permalink,
			"type":      "entry",
		}

		if comments[0].Author != "[deleted]" {
			data["author"] = map[string]interface{}{
				"name": comments[0].Author,
				"url":  "https://www.reddit.com/u/" + comments[0].Author,
				"type": "card",
			}
		}

		return data, nil
	}

	if strings.HasPrefix(id, "t3_") {
		if len(posts) != 1 {
			return nil, errors.New("post not found")
		}

		content := html.UnescapeString(posts[0].Body)
		if content == "[deleted]" || content == "" {
			content = posts[0].Title
		}

		if content == "[deleted]" {
			return nil, errors.New("post was deleted")
		}

		data := map[string]interface{}{
			"content":   content,
			"published": posts[0].Created.Time.Format(time.RFC3339),
			"url":       posts[0].URL,
			"type":      "entry",
		}

		if posts[0].Author != "[deleted]" {
			data["author"] = map[string]interface{}{
				"name": posts[0].Author,
				"url":  "https://www.reddit.com/u/" + posts[0].Author,
				"type": "card",
			}
		}

		return data, nil
	}

	return nil, errors.New("could not parse url")
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
	replyTo, err := urlpkg.Parse(urlStr)
	if err != nil {
		return "", err
	}

	path := strings.TrimSuffix(replyTo.Path, "/")
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
