package eagle

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

type ActivityPub struct {
	conf config.ActivityPub
	log  *zap.SugaredLogger

	privKey   crypto.PrivateKey
	signer    httpsig.Signer
	signerMu  sync.Mutex
	logMu     sync.Mutex
	followers *filemap

	// External services
	webmentions *Webmentions
	notify      *Notifications
}

func NewActivityPub(conf *config.Config, webmentions *Webmentions, notify *Notifications) (*ActivityPub, error) {
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

	followers, err := newFileMap(filepath.Join(conf.ActivityPub.Dir, "followers.json"))
	if err != nil {
		return nil, err
	}

	return &ActivityPub{
		conf:        conf.ActivityPub,
		log:         logging.S().Named("activitypub"),
		webmentions: webmentions,
		notify:      notify,
		privKey:     privateKey,
		signer:      signer,
		followers:   followers,
	}, nil
}

func (ap *ActivityPub) Create(activity map[string]interface{}) error {
	ap.log.Info("received create activity")
	object, exists := activity["object"].(map[string]interface{})
	if !exists {
		return fmt.Errorf("key 'object' not present or not map[string]interface{}: %v", object)
	}

	reply, hasReply := object["inReplyTo"].(string)
	id, hasID := object["id"].(string)

	if !hasReply || len(reply) == 0 {
		return fmt.Errorf("key 'inReplyTo' not present or not string: %v", reply)
	}

	if !hasID || len(id) == 0 {
		return fmt.Errorf("key 'id' not present or not string: %v", id)
	}

	if !strings.Contains(reply, ap.conf.IRI) {
		return fmt.Errorf("create activity destined to someone else: %s", reply)
	}

	err := ap.webmentions.SendWebmention(id, reply)
	if err != nil {
		return fmt.Errorf("could not convert activity to webmentions: %w", err)
	}

	return nil
}

func (ap *ActivityPub) Delete(activity map[string]interface{}) error {
	ap.log.Info("received delete activity")
	object, ok := activity["object"].(string)
	if !ok {
		return ErrNotHandled
	}

	if len(object) > 0 && activity["actor"] == object {
		err := ap.followers.remove(object)
		if err != nil {
			return fmt.Errorf("could not remove follower: %w", err)
		}
		return nil
	}

	return ErrNotHandled
}

func (ap *ActivityPub) Like(activity map[string]interface{}) error {
	// TODO: make new like and add it as webmention
	return ErrNotHandled
}

func (ap *ActivityPub) Follow(activity map[string]interface{}) error {
	ap.log.Info("received follow activity")
	iri, ok := activity["actor"].(string)
	if !ok || len(iri) == 0 {
		return fmt.Errorf("key 'actor' not present or not string: %v", iri)
	}

	if iri == activity["object"] {
		return nil
	}

	follower, err := ap.getActor(iri)
	if err != nil {
		return fmt.Errorf("failed to get actor %s: %w", iri, err)
	}

	if inbox, ok := ap.followers.get(follower.IRI); !ok || inbox != follower.Inbox {
		err = ap.followers.set(follower.IRI, follower.Inbox)
		if err != nil {
			return fmt.Errorf("failed to store followers: %w", err)
		}
	}

	delete(activity, "@context")
	accept := map[string]interface{}{}
	accept["@context"] = "https://www.w3.org/ns/activitystreams"
	accept["to"] = activity["actor"]
	accept["actor"] = ap.conf.IRI
	accept["object"] = activity
	accept["type"] = "Accept"
	_, accept["id"] = ap.newID()

	err = ap.sendSigned(accept, follower.Inbox)
	if err != nil {
		return fmt.Errorf("failed to send signed request: %w", err)
	}

	return nil
}

func (ap *ActivityPub) Undo(activity map[string]interface{}) error {
	ap.log.Info("received undo activity")
	object, ok := activity["object"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("key 'object' not present or not map[string]interface{}: %v: %w", object, ErrNotHandled)
	}

	objectType, ok := object["type"].(string)
	if !ok || objectType != "Follow" {
		return fmt.Errorf("key 'type' not present or not string: %v: %w", objectType, ErrNotHandled)
	}

	iri, ok := object["actor"].(string)
	if !ok || iri != activity["actor"] {
		ap.log.Debug("undo activity: object actor != activity actor, not handling")
		return fmt.Errorf("undo: object actor not activity actor: %v != %v: %w", iri, activity["actor"], ErrNotHandled)
	}

	return ap.followers.remove(iri)
}

func (ap *ActivityPub) Log(activity map[string]interface{}) error {
	ap.logMu.Lock()
	defer ap.logMu.Unlock()

	filename := filepath.Join(ap.conf.Dir, "log.json")
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

func (ap *ActivityPub) PostFollowers(activity map[string]interface{}) error {
	followers := ap.followers.getAll()
	id, ok := activity["id"].(string)
	if !ok {
		return fmt.Errorf("activity id %s must be string", id)
	}

	ap.log.Infof("sending create for %s", id)
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
		ap.log.Infof("sending announce for %s", id)
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
		ap.log.Infof("sending update for %s", id)
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
				ap.log.Errorw("could not send signed", "inbox", inbox, "activity", activity, "err", err)
				ap.notify.NotifyError(err)
			}
		}(followers[iri])
	}
}

func (ap *ActivityPub) sendSigned(b interface{}, to string) error {
	ap.log.Debugw("sending signed request", "to", to, "body", b)
	body, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("could not marshal activity: %w", err)
	}

	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, to, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	iri, err := url.Parse(to)
	if err != nil {
		return fmt.Errorf("could not parse iri: %w", err)
	}

	r.Header.Add("Accept-Charset", "utf-8")
	r.Header.Add("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	r.Header.Add("User-Agent", "hacdias.com")
	r.Header.Add("Accept", "application/activity+json; charset=utf-8")
	r.Header.Add("Content-Type", "application/activity+json; charset=utf-8")
	r.Header.Add("Host", iri.Host)

	ap.signerMu.Lock()
	err = ap.signer.SignRequest(ap.privKey, ap.conf.PubKeyID, r, bodyCopy)
	ap.signerMu.Unlock()
	if err != nil {
		return fmt.Errorf("could not sign request: %w", err)
	}

	ap.log.Debugw("sending request", "header", r.Header, "content", string(bodyCopy))
	resp, err := http.DefaultClient.Do(r)
	if !isSuccess(resp.StatusCode) {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("could not send signed request: failed with status %d: %s", resp.StatusCode, string(body))
	}
	return err
}

func (ap *ActivityPub) newID() (hash string, url string) {
	hash = uniuri.New()
	return hash, ap.conf.IRI + hash
}

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

type filemap struct {
	sync.RWMutex
	data map[string]string
	file string
}

func newFileMap(file string) (*filemap, error) {
	fm := &filemap{
		data: map[string]string{},
		file: file,
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fm, fm.save()
	} else if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &fm.data)
	if err != nil {
		return nil, err
	}

	return fm, nil
}

func (f *filemap) getAll() map[string]string {
	f.RLock()
	defer f.RUnlock()

	m := make(map[string]string)
	for key, value := range f.data {
		m[key] = value
	}

	return m
}

func (f *filemap) get(key string) (string, bool) {
	f.RLock()
	v, ok := f.data[key]
	f.RUnlock()
	return v, ok
}

func (f *filemap) remove(key string) error {
	f.Lock()
	defer f.Unlock()
	delete(f.data, key)
	return f.save()
}

func (f *filemap) set(key, value string) error {
	f.Lock()
	defer f.Unlock()
	f.data[key] = value
	return f.save()
}

func (f *filemap) save() error {
	bytes, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(f.file, bytes, 0644)
}
