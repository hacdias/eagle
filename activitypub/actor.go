package activitypub

import (
	"bytes"
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
	errActorStatusUnsuccessful = errors.New("actor could not be fetched")
	errActorNotFound           = errors.New("actor does not exist")
)

func (ap *ActivityPub) getRemoteActor(ctx context.Context, iri string) (typed.Typed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iri, new(bytes.Buffer))
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
			return nil, errActorNotFound
		}

		return nil, errActorStatusUnsuccessful
	}

	var actor typed.Typed
	err = json.NewDecoder(resp.Body).Decode(&actor)
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
