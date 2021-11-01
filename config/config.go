package config

import (
	urlpkg "net/url"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Development     bool
	Port            int
	BaseURL         string
	SourceDirectory string
	PublicDirectory string
	// WebhookSecret     string
	XRayEndpoint      string
	WebmentionsSecret string
	Auth              *Auth
	Tailscale         *Tailscale
	Tor               *Tor
	BunnyCDN          *BunnyCDN
	Telegram          *Telegram
	Twitter           *Twitter
	Miniflux          *Miniflux
	MeiliSearch       *MeiliSearch
}

// Parse parses the configuration from the default files and paths.
func Parse() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("port", 8080)
	viper.SetDefault("baseUrl", "http://localhost:8080")
	viper.SetDefault("sourceDirectory", "/app/source")
	viper.SetDefault("publicDirectory", "/app/public")
	viper.SetDefault("xrayEndpoint", "https://xray.p3k.app")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	conf.BaseURL, err = validateBaseURL(conf.BaseURL)
	if err != nil {
		return nil, err
	}

	conf.SourceDirectory, err = filepath.Abs(conf.SourceDirectory)
	if err != nil {
		return nil, err
	}

	conf.PublicDirectory, err = filepath.Abs(conf.PublicDirectory)
	if err != nil {
		return nil, err
	}

	if conf.Tor != nil {
		conf.Tor.ConfDir, err = filepath.Abs(conf.Tor.ConfDir)
		if err != nil {
			return nil, err
		}
	}

	return conf, nil
}

func validateBaseURL(url string) (string, error) {
	baseUrl, err := urlpkg.Parse(url)
	if err != nil {
		return "", err
	}

	baseUrl.Path = ""
	return baseUrl.String(), nil
}

type Auth struct {
	Username string
	Password string
	Secret   string
}

type Tailscale struct {
	ExclusiveDashboard bool
	Hostname           string
	Logging            bool
	Port               int
	AuthKey            string
}

type Tor struct {
	ConfDir string
	Logging bool
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
