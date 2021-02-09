package services

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/go-fed/httpsig"
	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/logging"
	"go.uber.org/zap"
)

var ErrNotHandled = errors.New("not handled")
var ErrNoChanges = errors.New("no changes")

type ActivityPub struct {
	*zap.SugaredLogger
	*Webmentions

	dir      string
	iri      string
	pubKeyId string
	privKey  crypto.PrivateKey
	signer   httpsig.Signer
	signerMu sync.Mutex
}

func NewActivityPub(conf *config.Config, webmentions *Webmentions) (*ActivityPub, error) {
	pkfile, err := ioutil.ReadFile(conf.ActivityPub.PrivKey)
	if err != nil {
		return nil, err
	}

	privateKeyDecoded, _ := pem.Decode(pkfile)
	if privateKeyDecoded == nil {
		return nil, err
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyDecoded.Bytes)
	if err != nil {
		return nil, err
	}

	prefs := []httpsig.Algorithm{httpsig.RSA_SHA256}
	digestAlgorithm := httpsig.DigestSha256
	headersToSign := []string{httpsig.RequestTarget, "date", "host", "digest"}
	signer, _, err := httpsig.NewSigner(prefs, digestAlgorithm, headersToSign, httpsig.Signature, 0)
	if err != nil {
		return nil, err
	}

	return &ActivityPub{
		SugaredLogger: logging.S().Named("activitypub"),
		Webmentions:   webmentions,
		dir:           filepath.Join(conf.Hugo.Source, "data", "activity"),
		iri:           conf.ActivityPub.IRI,
		pubKeyId:      conf.ActivityPub.PubKeyId,
		privKey:       privateKey,
		signer:        signer,
	}, nil
}

func (ap *ActivityPub) Create(activity map[string]interface{}) error {
	ap.Debug("processing create activity")
	object, exists := activity["object"].(map[string]interface{})
	if !exists {
		ap.Warn("create activity does not contain object")
		return errors.New("object must exist")
	}

	reply, hasReply := object["inReplyTo"].(string)
	id, hasID := object["id"].(string)

	if !hasReply || !hasID || len(reply) == 0 || len(id) == 0 {
		ap.Warn("create activity has invalid ID or inReplyTo")
		return errors.New("inReplyTo and id are required and need to be valid")
	}

	if !strings.Contains(reply, ap.iri) {
		ap.Warnf("create activity is destined to someone else: %s", reply)
		return fmt.Errorf("reply is not for me: %s", reply)
	}

	ap.Debug("converting create activity into webmention")
	err := ap.SendWebmention(id, reply)
	if err != nil {
		return err
	}
	return ErrNoChanges
}

func (ap *ActivityPub) Delete(activity map[string]interface{}) (string, error) {
	ap.Debug("received delete activity")
	object, ok := activity["object"].(string)
	if !ok {
		ap.Debug("delete activity not ok, not handlind")
		return "", ErrNotHandled
	}

	if len(object) > 0 && activity["actor"] == object {
		ap.Debugf("delete activity is unfollow from: %s", object)
		return object + " unfollowed you... 😔", ap.removeFollower(object)
	}

	ap.Debug("delete activity not ok, not handlind")
	return "", ErrNotHandled
}

func (ap *ActivityPub) removeFollower(iri string) error {
	ap.Debugf("removing follower %s", iri)
	followers, err := ap.Followers()
	if err != nil {
		return err
	}

	if _, ok := followers[iri]; !ok {
		return ErrNoChanges
	}

	delete(followers, iri)
	return ap.storeFollowers(followers)
}

func (ap *ActivityPub) Like(activity map[string]interface{}) (string, error) {
	// TODO: make new like and add it to mentions.json, send notification
	return "", ErrNotHandled
}

func (ap *ActivityPub) Follow(activity map[string]interface{}) (string, error) {
	ap.Debug("received follow activity")
	iri, ok := activity["actor"].(string)
	if !ok || len(iri) == 0 {
		ap.Debugw("activity has no actor", "activity", activity)
		return "", errors.New("actor should exist in activity")
	}

	if iri == activity["object"] {
		// Avoid following myself. Why would this happen though?
		return "", nil
	}

	follower, err := ap.getActor(iri)
	if err != nil {
		ap.Debugf("failed to get actor %s: %s", iri, err)
		return "", err
	}

	followers, err := ap.Followers()
	if err != nil {
		ap.Debugf("failed to get followers: %s", err)
		return "", err
	}

	changed := false
	if inbox, ok := followers[follower.IRI]; !ok || inbox != follower.Inbox {
		followers[follower.IRI] = follower.Inbox
		changed = true

		err = ap.storeFollowers(followers)
		if err != nil {
			ap.Debugf("failed to store followers: %s", err)
			return "", err
		}
	}

	delete(activity, "@context")
	accept := map[string]interface{}{}
	accept["@context"] = "https://www.w3.org/ns/activitystreams"
	accept["to"] = activity["actor"]
	accept["actor"] = ap.iri
	accept["object"] = activity
	accept["type"] = "Accept"
	_, accept["id"] = ap.newID()

	err = ap.sendSigned(accept, follower.Inbox)
	if err != nil {
		ap.Debugf("failed to send signed request: %s", err)
		return "", err
	}

	if !changed {
		err = ErrNoChanges
	}

	return follower.IRI + " followed you! 🤯", err
}

func (ap *ActivityPub) Undo(activity map[string]interface{}) (string, error) {
	ap.Info("received undo activity")
	object, ok := activity["object"].(map[string]interface{})
	if !ok {
		ap.Debug("undo activity: object not ok, not handling")
		return "", ErrNotHandled
	}

	objectType, ok := object["type"].(string)
	if !ok || objectType != "Follow" {
		ap.Debug("undo activity: object type not supported, not handling")
		return "", ErrNotHandled
	}

	iri, ok := object["actor"].(string)
	if !ok || iri != activity["actor"] {
		ap.Debug("undo activity: object actor != activity actor, not handling")
		return "", ErrNotHandled
	}

	ap.Infof("undo activity: unfollowed by %s", iri)
	return iri + " unfollowed you... 😔", ap.removeFollower(iri)
}

func (ap *ActivityPub) Followers() (map[string]string, error) {
	fd, err := os.Open(filepath.Join(ap.dir, "followers.json"))
	if err != nil {
		if os.IsNotExist(err) {
			err = ap.storeFollowers(map[string]string{})
			return map[string]string{}, err
		}

		return nil, err
	}
	defer fd.Close()

	var f map[string]string
	return f, json.NewDecoder(fd).Decode(&f)
}

func (ap *ActivityPub) Log(activity map[string]interface{}) error {
	filename := filepath.Join(ap.dir, "log.json")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	bytes, err := json.Marshal(activity)
	if err != nil {
		return err
	}

	bytes = append(bytes, '\n')

	_, err = f.Write(bytes)
	return err
}

func (ap *ActivityPub) storeFollowers(f map[string]string) error {
	bytes, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(ap.dir, "followers.json"), bytes, 0644)
}

func (ap *ActivityPub) PostFollowers(activity map[string]interface{}) error {
	followers, err := ap.Followers()
	if err != nil {
		return err
	}

	id, ok := activity["id"].(string)
	if !ok {
		return fmt.Errorf("activity id %s must be string", id)
	}

	ap.Infof("sending create for %s", id)
	create := make(map[string]interface{})
	create["@context"] = []string{"https://www.w3.org/ns/activitystreams"}
	create["type"] = "Create"
	create["actor"] = activity["attributedTo"]
	create["id"] = id
	create["to"] = activity["to"]
	create["published"] = activity["published"]
	create["object"] = activity
	ap.sendTo(create, followers)

	// Boost if it contains "inReplyTo"
	if activity["inReplyTo"] != nil {
		ap.Infof("sending announce for %s", id)
		announce := make(map[string]interface{})
		announce["@context"] = []string{"https://www.w3.org/ns/activitystreams"}
		announce["type"] = "Announce"
		announce["id"] = id + "#announce"
		announce["actor"] = activity["attributedTo"]
		announce["to"] = activity["to"]
		announce["published"] = activity["published"]
		announce["object"] = id
		ap.sendTo(announce, followers)
	}

	// Send an update event if it contains "updated" and "updated" !== "published"
	if activity["updated"] != nil && activity["published"] != nil && activity["updated"] != activity["published"] {
		ap.Infof("sending update for %s", id)
		update := make(map[string]interface{})
		update["@context"] = []string{"https://www.w3.org/ns/activitystreams"}
		update["type"] = "Update"
		update["actor"] = activity["attributedTo"]
		update["to"] = activity["to"]
		update["published"] = activity["published"]
		update["updated"] = activity["updated"]
		update["object"] = activity
		ap.sendTo(update, followers)
	}

	return nil
}

func (ap *ActivityPub) sendTo(activity map[string]interface{}, followers map[string]string) {
	for iri := range followers {
		go func(inbox string) {
			err := ap.sendSigned(activity, inbox)
			if err != nil {
				ap.Errorw("could not send signed", "inbox", inbox, "activity", activity)
			}
		}(followers[iri])
	}
}

func (ap *ActivityPub) sendSigned(b interface{}, to string) error {
	ap.Debugw("sending signed request", "to", to, "body", b)
	body, err := json.Marshal(b)
	if err != nil {
		return err
	}

	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, to, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	iri, err := url.Parse(to)
	if err != nil {
		return err
	}

	r.Header.Add("Accept-Charset", "utf-8")
	r.Header.Add("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	r.Header.Add("User-Agent", "hacdias.com")
	r.Header.Add("Accept", "application/activity+json; charset=utf-8")
	r.Header.Add("Content-Type", "application/activity+json; charset=utf-8")
	r.Header.Add("Host", iri.Host)

	ap.signerMu.Lock()
	err = ap.signer.SignRequest(ap.privKey, ap.pubKeyId, r, bodyCopy)
	ap.signerMu.Unlock()
	if err != nil {
		return err
	}

	ap.Debugw("sending request", "header", r.Header, "content", string(bodyCopy))

	resp, err := http.DefaultClient.Do(r)
	if !isSuccess(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("signed request failed with status %d: %s", resp.StatusCode, string(body))
	}
	return err
}

func (ap *ActivityPub) newID() (hash string, url string) {
	hash = uniuri.New()
	return hash, ap.iri + hash
}
