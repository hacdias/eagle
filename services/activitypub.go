package services

import (
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-fed/httpsig"
)

var ErrNotHandled = errors.New("not handled")

type ActivityPub struct {
	IRI         string
	pubKeyId    string
	privKey     crypto.PrivateKey
	Webmentions *Webmentions
}

func (ap *ActivityPub) Create(activity map[string]interface{}) error {
	object, exists := activity["object"].(map[string]interface{})
	if !exists {
		return errors.New("object must exist")
	}

	reply, hasReply := object["inReplyTo"].(string)
	id, hasID := object["id"].(string)

	if !hasReply || !hasID || len(reply) == 0 || len(id) == 0 {
		return errors.New("inReplyTo and id are required and need to be valid")
	}

	if strings.Contains(reply, ap.IRI) {
		return fmt.Errorf("reply is not for me: %s", reply)
	}

	return ap.Webmentions.Send(id, reply)
}

func (ap *ActivityPub) Delete(activity map[string]interface{}) error {
	// TODO: based on @jlese's code, try to remove follower
	// if object, ok := activity["object"].(string); ok && len(object) > 0 && activity["actor"] == object {
	// 	_ = actor.RemoveFollower(object)
	// }
	return ErrNotHandled
}

func (ap *ActivityPub) Like(activity map[string]interface{}) error {
	// TODO: make new like and add it to mentions.json, send notification
	return ErrNotHandled
}

func (ap *ActivityPub) Follow(activity map[string]interface{}) error {

	/*
		// it's a follow, write it down
		newFollower := follow["actor"].(string)
		fmt.Println("New follow request:", newFollower)
		// check we aren't following ourselves
		if newFollower == follow["object"] {
			// actor and object are equal
			return
		}
		follower, err := NewRemoteActor(newFollower)
		if err != nil {
			// Couldn't retrieve remote actor info
			fmt.Println("Failed to retrieve remote actor info:", newFollower)
			return
		}
		// Add or update follower
		_ = a.NewFollower(newFollower, follower.inbox)
		// remove @context from the inner activity
		delete(follow, "@context")
		accept := make(map[string]interface{})
		accept["@context"] = "https://www.w3.org/ns/activitystreams"
		accept["to"] = follow["actor"]
		_, accept["id"] = a.newID()
		accept["actor"] = a.iri
		accept["object"] = follow
		accept["type"] = "Accept"
		err = a.signedHTTPPost(accept, follower.inbox)
		if err != nil {
			fmt.Println("Failed to accept:", follower.iri)
			fmt.Println(err.Error())
		} else {
			fmt.Println("Accepted:", follower.iri)
			if telegramBot != nil {
				_ = telegramBot.Post(follower.iri + " followed")
			}
		}
	*/

	// We could pass an error back up, if desired.
	return ErrNotHandled
}

func (ap *ActivityPub) Undo(activity map[string]interface{}) error {
	// TODO: based on @jlelse's code, tru to remove follower
	// if object, ok := activity["object"].(map[string]interface{}); ok {
	// 	if objectType, ok := object["type"].(string); ok && objectType == "Follow" {
	// 		if iri, ok := object["actor"].(string); ok && iri == activity["actor"] {
	// 			_ = actor.RemoveFollower(iri)
	// 			fmt.Println(iri, "unfollowed")
	// 			if telegramBot != nil {
	// 				_ = telegramBot.Post(iri + " unfollowed")
	// 			}
	// 		}
	// 	}
	// }

	return ErrNotHandled
}

func (ap *ActivityPub) Followers() ([]string, error) {
	return []string{}, nil
}

func (ap *ActivityPub) PostFollowers(activity map[string]interface{}) error {
	/*

		https://git.jlel.se/jlelse/jsonpub/src/branch/master/actor.go#L112
	*/

	return nil
}

func (ap *ActivityPub) sendSigned(b interface{}, r *http.Request) error {
	prefs := []httpsig.Algorithm{httpsig.RSA_SHA256}
	digestAlgorithm := httpsig.DigestSha256

	r.Header.Add("Accept-Charset", "utf-8")
	r.Header.Add("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	r.Header.Add("Accept", "application/activity+json; charset=utf-8")
	r.Header.Add("Content-Type", "application/activity+json; charset=utf-8")
	r.Header.Add("Host", ap.IRI)

	headersToSign := []string{httpsig.RequestTarget, "date", "host", "digest"}
	signer, _, err := httpsig.NewSigner(prefs, digestAlgorithm, headersToSign, httpsig.Signature)
	if err != nil {
		return err
	}

	body, err := json.Marshal(b)
	if err != nil {
		return err
	}

	err = signer.SignRequest(ap.privKey, ap.pubKeyId, r, body)
	if err != nil {
		return err
	}

	_, err = http.DefaultClient.Do(r)
	return err
}
