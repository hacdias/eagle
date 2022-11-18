package activitypub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

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

func (ap *ActivityPub) getActor(ctx context.Context, iri string) (typed.Typed, error) {
	actor, err := ap.getActivity(ctx, iri)
	if err != nil {
		return nil, err
	}

	if t := actor.String("type"); t != "Person" {
		return nil, fmt.Errorf("actor %s type should be Person, received %s", iri, t)
	}

	if actor.String("id") == "" {
		return nil, fmt.Errorf("actor %s has invalid id", iri)
	}

	return actor, nil
}

func (ap *ActivityPub) getActorFromActivity(ctx context.Context, url string) (typed.Typed, error) {
	activity, err := ap.getActivity(ctx, url)
	if err != nil {
		return nil, err
	}

	iri, ok := activity.StringIf("attributedTo")
	if !ok || len(iri) == 0 {
		return nil, errors.New("attributedTo field is empty")
	}

	return ap.getActor(ctx, iri)
}
