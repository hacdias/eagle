package config

import (
	"net/url"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Development bool

	Website   Server
	Dashboard Server
	Auth      *Auth

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

	viper.SetDefault("website.port", 8080)
	viper.SetDefault("website.baseUrl", "http://localhost:8080")

	viper.SetDefault("dashboard.port", 8081)
	viper.SetDefault("dashboard.baseUrl", "http://localhost:8081")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	conf.Website.BaseURL, err = validateBaseURL(conf.Website.BaseURL)
	if err != nil {
		return nil, err
	}

	conf.Dashboard.BaseURL, err = validateBaseURL(conf.Dashboard.BaseURL)
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

	return conf, nil
}

func validateBaseURL(s string) (string, error) {
	baseUrl, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	baseUrl.Path = ""
	return baseUrl.String(), nil
}

type Server struct {
	Port    int
	BaseURL string
}

func (s *Server) IsHTTPS() bool {
	u, _ := url.Parse(s.BaseURL)
	return u.Scheme == "https"
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
