package config

import (
	"net/url"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Development bool
	Domain      string

	WebsitePort   int
	DashboardPort int
	Auth          Auth // TODO: make optional

	Webmentions Webmentions
	Webhook     Webhook
	Hugo        Hugo
	XRay        XRay
	Telegram    Telegram
	BunnyCDN    BunnyCDN

	Twitter     *Twitter
	Miniflux    *Miniflux
	MeiliSearch *MeiliSearch
}

// Parse parses the configuration from the default files and paths.
func Parse() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("websitePort", 8080)
	viper.SetDefault("dashboardPort", 8081)
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

type Auth struct {
	Username string
	Password string
	Secret   string
}

type Webmentions struct {
	TelegraphToken string
	Secret         string
}

type Webhook struct {
	Secret string
}

type Hugo struct {
	Source      string
	Destination string
}

type XRay struct {
	Endpoint string
}
type Telegram struct {
	Token  string
	ChatID int64
}

type BunnyCDN struct {
	Zone string
	Key  string
	Base string
}

type Twitter struct {
	User        string
	Key         string
	Secret      string
	Token       string
	TokenSecret string
}

type Miniflux struct {
	Endpoint string
	Key      string
}

type MeiliSearch struct {
	Endpoint string
	Key      string
}
