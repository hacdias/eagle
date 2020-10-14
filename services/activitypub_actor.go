package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type actor struct {
	IRI   string
	Inbox string
}

func (ap *ActivityPub) getActor(url string) (*actor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	buf := new(bytes.Buffer)
	req, err := http.NewRequestWithContext(ctx, "GET", url, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/activity+json")
	req.Header.Add("Accept-Charset", "utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if !isSuccess(resp.StatusCode) {
		return nil, fmt.Errorf("request was not successfull: code %d", resp.StatusCode)
	}

	var e map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&e)
	if err != nil {
		return nil, err
	}

	if e["type"] != "Person" {
		return nil, fmt.Errorf("actor %s should be a Person, received %s", url, e["type"])
	}

	iri, iriOK := e["id"].(string)
	inbox, inboxOK := e["inbox"].(string)

	if !iriOK || !inboxOK || len(iri) == 0 || len(inbox) == 0 {
		return nil, fmt.Errorf("actor %s has wrong iri or inbox: %s, %s", url, iri, inbox)
	}

	return &actor{
		IRI:   iri,
		Inbox: inbox,
	}, nil
}

func isSuccess(code int) bool {
	return code == http.StatusOK ||
		code == http.StatusCreated ||
		code == http.StatusAccepted ||
		code == http.StatusNoContent
}
