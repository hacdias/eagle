package syndicator

import (
	"errors"
	"fmt"
	"net/http"
	urlpkg "net/url"
	"strings"

	"github.com/hacdias/eagle/v3/config"
	"github.com/hacdias/eagle/v3/entry"
	"github.com/hacdias/eagle/v3/entry/mf2"
)

type Reddit struct {
	conf   *config.Reddit
	client *http.Client
}

func NewReddit(opts *config.Twitter) *Reddit {
	// config := oauth1.NewConfig(opts.Key, opts.Secret)
	// token := oauth1.NewToken(opts.Token, opts.TokenSecret)

	// client := config.Client(oauth1.NoContext, token)
	// client.Timeout = time.Second * 30

	// return &Reddit{
	// 	conf:   opts,
	// 	client: client,
	// }
	return nil
}

func (r *Reddit) Syndicate(entry *entry.Entry) (url string, err error) {

	// Like -> Upvote https://www.reddit.com/dev/api#POST_api_vote
	// Reply -> Reply https://www.reddit.com/dev/api#POST_api_comment
	// Others -> New Post ^Same?

	return "", errors.New("not implemented")
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
	return fmt.Sprintf("Reddit (%s)", r.conf.User)
}

func (r *Reddit) Identifier() string {
	return fmt.Sprintf("Reddit-%s", r.conf.User)
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
