package activitypub

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/hacdias/eagle/pkg/contenttype"
	"github.com/karlseguin/typed"
)

func generateKeyPair(privKeyFilename, pubKeyFilename string) error {
	privKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	privKeyFile, err := os.OpenFile(privKeyFilename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer privKeyFile.Close()

	err = pem.Encode(privKeyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})
	if err != nil {
		return err
	}

	publicKeyFile, err := os.OpenFile(pubKeyFilename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer publicKeyFile.Close()

	pubKey, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return err
	}

	err = pem.Encode(publicKeyFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKey,
	})
	if err != nil {
		return err
	}

	return nil
}

func getKeyPair(directory string) (*rsa.PrivateKey, string, error) {
	privKeyFilename := filepath.Join(directory, "private.key")
	pubKeyFilename := filepath.Join(directory, "public.key")

	_, err := os.Stat(privKeyFilename)
	if err != nil {
		if os.IsNotExist(err) {
			err = generateKeyPair(privKeyFilename, pubKeyFilename)
			if err != nil {
				return nil, "", err
			}
		} else {
			return nil, "", err
		}
	}

	privateKeyBytes, err := os.ReadFile(privKeyFilename)
	if err != nil {
		return nil, "", err
	}

	publicKeyBytes, err := os.ReadFile(pubKeyFilename)
	if err != nil {
		return nil, "", err
	}

	privKeyDecoded, _ := pem.Decode(privateKeyBytes)
	if privKeyDecoded == nil {
		return nil, "", errors.New("cannot decode private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privKeyDecoded.Bytes)
	return privateKey, string(publicKeyBytes), err
}

func getSigner() (httpsig.Signer, error) {
	algorithms := []httpsig.Algorithm{httpsig.RSA_SHA256}
	digestAlgorithm := httpsig.DigestSha256
	headersToSign := []string{httpsig.RequestTarget, "date", "host", "digest"}
	signer, _, err := httpsig.NewSigner(algorithms, digestAlgorithm, headersToSign, httpsig.Signature, 0)
	return signer, err
}

func (ap *ActivityPub) verifySignature(r *http.Request) (typed.Typed, string, error) {
	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		return nil, "", err
	}

	keyID := verifier.KeyId()
	actor, err := ap.getActor(r.Context(), keyID)
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

func (ap *ActivityPub) sendSigned(ctx context.Context, activity interface{}, inbox string) error {
	body, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("could not marshal data: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, inbox, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	r.Header.Add("Accept", contenttype.ASUTF8)
	r.Header.Add("Accept-Charset", "utf-8")
	r.Header.Add("Content-Type", contenttype.ASUTF8)
	r.Header.Add("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	r.Header.Add("User-Agent", ap.c.Server.BaseURL)
	r.Header.Add("Host", r.URL.Host)

	ap.signerMu.Lock()
	err = ap.signer.SignRequest(ap.privKey, ap.getSelfKeyID(), r, body)
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
			string(body),
		)
	}

	return nil
}
