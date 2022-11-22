package activitypub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/pkg/contenttype"
	"github.com/karlseguin/typed"
)

var (
	errStatusUnsuccessful = errors.New("activity could not be fetched")
	errNotFound           = errors.New("activity does not exist")
)

func (ap *ActivityPub) getActivity(ctx context.Context, url string) (typed.Typed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", contenttype.AS)
	req.Header.Add("Accept-Charset", "utf-8")
	req.Header.Set("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("User-Agent", ap.c.Server.BaseURL)

	ap.signerMu.Lock()
	err = ap.signer.SignRequest(ap.privKey, ap.getSelfKeyID(), req, []byte(""))
	ap.signerMu.Unlock()
	if err != nil {
		return nil, err
	}

	resp, err := ap.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !isSuccess(resp.StatusCode) {
		if isDeleted(resp.StatusCode) {
			return nil, errNotFound
		}

		return nil, errStatusUnsuccessful
	}

	var activity typed.Typed
	err = json.NewDecoder(resp.Body).Decode(&activity)
	if err != nil {
		return nil, err
	}

	return activity, nil
}

func (ap *ActivityPub) getActorByIRI(ctx context.Context, iri string) (typed.Typed, error) {
	if !strings.Contains(iri, "@") {
		return nil, fmt.Errorf("%s does not contain a domain", iri)
	}

	domain := strings.SplitN(iri, "@", 2)[1]

	webfinger, err := ap.getWebfinger(ctx, domain, "acct:"+iri)
	if err != nil {
		return nil, err
	}

	for _, link := range webfinger.Links {
		if link.Rel == "self" && strings.Contains(link.Type, contenttype.AS) {
			return ap.getActorByID(ctx, link.Href)
		}
	}

	return nil, errors.New("actor not found")
}

func (ap *ActivityPub) getActorByID(ctx context.Context, id string) (typed.Typed, error) {
	actor, err := ap.getActivity(ctx, id)
	if err != nil {
		return nil, err
	}

	if t := actor.String("type"); t != "Person" {
		return nil, fmt.Errorf("actor %s type should be Person, received %s", id, t)
	}

	if actor.String("id") == "" {
		return nil, fmt.Errorf("actor %s has invalid id", id)
	}

	return actor, nil
}

func (ap *ActivityPub) getActorFromActivity(ctx context.Context, url string) (typed.Typed, typed.Typed, error) {
	activity, err := ap.getActivity(ctx, url)
	if err != nil {
		return nil, nil, err
	}

	var iri string
	if v, ok := activity.StringIf("attributedTo"); ok && v != "" {
		iri = v
	} else if v, ok := activity.ObjectIf("attributedTo"); ok {
		iri = v.String("id")
	}

	if iri == "" {
		return nil, nil, errors.New("attributedTo field is empty or not string")
	}

	actor, err := ap.getActorByID(ctx, iri)
	if err != nil {
		return nil, nil, err
	}

	return actor, activity, nil
}

func (ap *ActivityPub) getWebfinger(ctx context.Context, domain, resource string) (*eagle.WebFinger, error) {
	url := fmt.Sprintf("https://%s/.well-known/webfinger?resource=%s", domain, resource)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ap.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var webfinger *eagle.WebFinger
	err = json.NewDecoder(resp.Body).Decode(&webfinger)
	if err != nil {
		return nil, err
	}

	return webfinger, nil
}
