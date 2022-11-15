package activitypub

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/hacdias/eagle/pkg/contenttype"
	"github.com/karlseguin/typed"
)

func (ap *ActivityPub) verifySignature(r *http.Request) (typed.Typed, string, error) {
	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		return nil, "", err
	}

	keyID := verifier.KeyId()
	actor, err := ap.getRemoteActor(r.Context(), keyID)
	if err != nil {
		return nil, "", err
	}

	publicKey, ok := actor.ObjectIf("publicKey")
	if !ok {
		return nil, "", errors.New("actor has no public key")
	}

	publicKeyPem := publicKey.String("publicKeyPem")
	if publicKeyPem == "" {
		return nil, "", errors.New("actor has no public key")
	}

	block, _ := pem.Decode([]byte(publicKeyPem))
	if block == nil {
		return nil, keyID, errors.New("public key invalid")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Unable to parse public key
		return nil, keyID, err
	}
	return actor, keyID, verifier.Verify(pubKey, httpsig.RSA_SHA256)
}

func (ap *ActivityPub) send(ctx context.Context, activity interface{}, inbox string) error {
	body, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("could not marshal data: %w", err)
	}

	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, inbox, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	iri, err := url.Parse(inbox)
	if err != nil {
		return fmt.Errorf("could not parse iri: %w", err)
	}

	r.Header.Add("Accept-Charset", "utf-8")
	r.Header.Add("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	r.Header.Add("User-Agent", ap.c.Server.BaseURL)
	r.Header.Add("Accept", contenttype.ASUTF8)
	r.Header.Add("Content-Type", contenttype.ASUTF8)
	r.Header.Add("Host", iri.Host)

	ap.signerMu.Lock()
	err = ap.signer.SignRequest(ap.privKey, ap.getSelfKeyID(), r, bodyCopy)
	ap.signerMu.Unlock()
	if err != nil {
		return fmt.Errorf("could not sign request: %w", err)
	}

	resp, err := ap.httpClient.Do(r)
	if err != nil {
		return err
	}

	if !isSuccess(resp.StatusCode) {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"could not send signed request: failed with status %d\n\nBody:\n%s\n\nHeaders:\n%v\n\nContent:\n%s",
			resp.StatusCode,
			string(body),
			r.Header,
			string(bodyCopy),
		)
	}

	return nil
}
