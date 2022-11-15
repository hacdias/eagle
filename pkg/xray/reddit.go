package xray

import (
	"context"
	"errors"
	"html"
	urlpkg "net/url"
	"strings"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

var (
	ErrRedditCommentNotFound = errors.New("reddit comment not found")
	ErrRedditPostNotFound    = errors.New("reddit post not found")
	ErrRedditPostDeleted     = errors.New("reddit post deleted")
	ErrUnsupportedRedditURL  = errors.New("unsupported reddit url")
)

func (x *XRay) fetchAndParseRedditURL(urlStr string) (*Post, interface{}, error) {
	url, err := urlpkg.Parse(urlStr)
	if err != nil {
		return nil, nil, err
	}

	path := strings.TrimSuffix(url.Path, "/")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 5 {
		return x.fetchAndParseRedditPost("t3_" + parts[3])
	} else if len(parts) == 6 {
		return x.fetchAndParseRedditComment("t1_" + parts[5])
	}

	return nil, nil, ErrUnsupportedRedditURL
}

func (x *XRay) fetchRedditComment(id string) (*reddit.Comment, error) {
	_, comments, _, _, err := x.redditClient.Listings.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	if len(comments) != 1 {
		return nil, ErrRedditCommentNotFound
	}

	return comments[0], nil
}

func (x *XRay) parseRedditComment(comment *reddit.Comment) (*Post, error) {
	content := html.UnescapeString(comment.Body)
	if content == "[deleted]" {
		return nil, errors.New("comment was deleted")
	}

	parsed := &Post{
		Content:   SanitizeContent(content),
		Published: comment.Created.Time,
		URL:       "https://www.reddit.com" + comment.Permalink,
	}

	if comment.Author != "[deleted]" {
		parsed.Author.Name = comment.Author
		parsed.Author.URL = "https://www.reddit.com/u/" + comment.Author
	}

	return parsed, nil
}

func (x *XRay) fetchAndParseRedditComment(id string) (*Post, interface{}, error) {
	raw, err := x.fetchRedditComment(id)
	if err != nil {
		return nil, nil, err
	}

	post, err := x.parseRedditComment(raw)
	if err != nil {
		return nil, nil, err
	}

	return post, raw, nil
}

func (x *XRay) fetchRedditPost(postID string) (*reddit.Post, error) {
	posts, _, _, _, err := x.redditClient.Listings.Get(context.Background(), postID)
	if err != nil {
		return nil, err
	}

	if len(posts) != 1 {
		return nil, ErrRedditPostNotFound
	}

	return posts[0], nil
}

func (x *XRay) parseRedditPost(post *reddit.Post) (*Post, error) {
	content := html.UnescapeString(post.Body)
	if content == "[deleted]" || content == "" {
		content = post.Title
	}

	if content == "[deleted]" {
		return nil, ErrRedditPostDeleted
	}

	parsed := &Post{
		Content:   SanitizeContent(content),
		Published: post.Created.Time,
		URL:       post.URL,
	}

	if post.Author != "[deleted]" {
		parsed.Author.Name = post.Author
		parsed.Author.URL = "https://www.reddit.com/u/" + post.Author
	}

	return parsed, nil
}

func (x *XRay) fetchAndParseRedditPost(id string) (*Post, interface{}, error) {
	raw, err := x.fetchRedditPost(id)
	if err != nil {
		return nil, nil, err
	}

	post, err := x.parseRedditPost(raw)
	if err != nil {
		return nil, nil, err
	}

	return post, raw, nil
}
