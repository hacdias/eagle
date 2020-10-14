package services

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/hacdias/eagle/config"
)

type Services struct {
	PublicDirChanges chan string
	cfg              *config.Config
	Store            StorageService
	Hugo             *Hugo
	Media            *Media
	Notify           *Notify
	Webmentions      *Webmentions
	XRay             *XRay
	Syndicator       Syndicator
	MeiliSearch      *MeiliSearch
	ActivityPub      *ActivityPub
}

func NewServices(c *config.Config) (*Services, error) {
	notify, err := NewNotify(&c.Telegram, c.S().Named("telegram"))
	if err != nil {
		return nil, err
	}

	var store StorageService
	if c.Development {
		store = &PlaceboStorage{}
	} else {
		store = &GitStorage{
			Directory: c.Hugo.Source,
		}
	}

	dirChanges := make(chan string)

	hugo := &Hugo{
		Hugo:       c.Hugo,
		Domain:     c.Domain,
		DirChanges: dirChanges,
	}

	media := &Media{c.BunnyCDN}

	webmentions := &Webmentions{
		SugaredLogger: c.S().Named("webmentions"),
		Domain:        c.Domain,
		Telegraph:     c.Telegraph,
		Hugo:          hugo,
		Media:         media,
	}

	privateKey, err := decodePrivateKey(c.ActivityPub.PrivKey)
	if err != nil {
		return nil, err
	}

	activitypub := &ActivityPub{
		SugaredLogger: c.S().Named("activitypub"),
		Dir:           filepath.Join(c.Hugo.Source, "data", "activity"),
		IRI:           c.ActivityPub.IRI,
		pubKeyId:      c.ActivityPub.PubKeyId,
		privKey:       privateKey,
		Webmentions:   webmentions,
	}

	syndicator := Syndicator{}

	if c.Twitter.User != "" {
		syndicator["https://twitter.com/"+c.Twitter.User] = NewTwitter(&c.Twitter)
	}

	services := &Services{
		PublicDirChanges: dirChanges,
		cfg:              c,
		Store:            store,
		Hugo:             hugo,
		Media:            media,
		Notify:           notify,
		Webmentions:      webmentions,
		XRay: &XRay{
			SugaredLogger: c.S().Named("xray"),
			XRay:          c.XRay,
			Twitter:       c.Twitter,
			StoragePath:   path.Join(c.Hugo.Source, "data", "xray"),
		},
		Syndicator:  syndicator,
		ActivityPub: activitypub,
	}

	if c.MeiliSearch != nil {
		services.MeiliSearch, err = NewMeiliSearch(c.MeiliSearch)
		if err != nil {
			return nil, err
		}
	}

	return services, nil
}

func decodePrivateKey(path string) (*rsa.PrivateKey, error) {
	pkfile, err := ioutil.ReadFile(path)
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

	return privateKey, nil
}
