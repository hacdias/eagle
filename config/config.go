package config

import (
	"net/url"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Port         int
	PortAdmin    int
	Domain       string
	Development  bool
	Telegraph    Telegraph
	XRay         XRay
	Hugo         Hugo
	Twitter      Twitter
	Telegram     Telegram
	BunnyCDN     BunnyCDN
	WebmentionIO WebmentionIO
	Webhook      Webhook
	MeiliSearch  *MeiliSearch
	Auth         Auth
}

// Parse parses the configuration from the default files and paths.
func Parse() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("port", 8080)
	viper.SetDefault("portAdmin", 8081)
	viper.SetDefault("domain", "http://localhost:8080")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	domain, err := url.Parse(conf.Domain)
	if err != nil {
		return nil, err
	}

	conf.Hugo.Source, err = filepath.Abs(conf.Hugo.Source)
	if err != nil {
		return nil, err
	}

	conf.Hugo.Destination, err = filepath.Abs(conf.Hugo.Destination)
	if err != nil {
		return nil, err
	}

	domain.Path = ""
	conf.Domain = domain.String()
	return conf, nil
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

type Auth struct {
	Username string
	Password string
	Secret   string
}
