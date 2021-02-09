package config

import "go.uber.org/zap"

type Config struct {
	logger       *zap.Logger `mapstructure:"-"`
	Port         int
	Domain       string // MUST NOT CONTAIN END SLASH
	Development  bool
	Telegraph    Telegraph
	XRay         XRay
	Hugo         Hugo
	Twitter      Twitter
	Telegram     Telegram
	BunnyCDN     BunnyCDN
	WebmentionIO WebmentionIO
	Webhook      Webhook
	ActivityPub  ActivityPub
	MeiliSearch  *MeiliSearch
	BasicAuth    map[string]string
}

func (c *Config) S() *zap.SugaredLogger {
	return c.logger.Sugar()
}

func (c *Config) L() *zap.Logger {
	return c.logger
}

type Twitter struct {
	User        string
	Key         string
	Secret      string
	Token       string
	TokenSecret string `mapstructure:"token_secret"`
}

type Telegram struct {
	Token  string
	ChatID int64 `mapstructure:"chat_id"`
}

type BunnyCDN struct {
	Zone string
	Key  string
	Base string
}

type WebmentionIO struct {
	Secret string
}

type Webhook struct {
	Secret string
}

type Hugo struct {
	Source      string
	Destination string
}

type Telegraph struct {
	Token string
}

type XRay struct {
	Endpoint string
}

type MeiliSearch struct {
	Endpoint string
	Key      string
}

type ActivityPub struct {
	IRI      string
	PubKeyId string `mapstructure:"pub_key_id"`
	PrivKey  string `mapstructure:"priv_key"`
}
