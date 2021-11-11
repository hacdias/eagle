package config

import (
	"fmt"
	urlpkg "net/url"
	"path/filepath"

	"github.com/hacdias/eagle/v2/entry/mf2"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
)

type Config struct {
	Development     bool
	Port            int
	SourceDirectory string
	PublicDirectory string
	Site            Site
	User            User
	// WebhookSecret     string
	XRayEndpoint      string
	WebmentionsSecret string
	Auth              Auth
	PostgreSQL        PostgreSQL
	Tor               *Tor
	BunnyCDN          *BunnyCDN
	Telegram          *Telegram
	Twitter           *Twitter
	Miniflux          *Miniflux
}

// Parse parses the configuration from the default files and paths.
func Parse() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	viper.SetDefault("port", 8080)
	viper.SetDefault("sourceDirectory", "/app/source")
	viper.SetDefault("publicDirectory", "/app/public")
	viper.SetDefault("xrayEndpoint", "https://xray.p3k.app")

	viper.SetDefault("site.baseUrl", "http://localhost:8080")
	viper.SetDefault("site.language", "en")
	viper.SetDefault("site.paginate", 15)

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = viper.Unmarshal(conf)
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

	for typ, sections := range conf.Site.MicropubTypes {
		if !mf2.IsType(typ) {
			return nil, fmt.Errorf("%s is not a valid micropub type", typ)
		}

		conf.Site.Sections = append(conf.Site.Sections, sections...)
	}

	conf.Site.Sections = funk.UniqString(conf.Site.Sections)

	conf.Site.BaseURL, err = validateBaseURL(conf.Site.BaseURL)
	if err != nil {
		return nil, err
	}

	// TODO(future): add more thorough verification.
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

type Site struct {
	Language      string
	Title         string
	BaseURL       string
	Emoji         string
	Description   string
	Sections      []string
	MicropubTypes map[mf2.Type][]string
	IndexSections []string
	Paginate      int
	Menus         map[string][]MenuItem
}

type MenuItem struct {
	Name  string
	Emoji string
	Link  string
}

type User struct {
	Name     string
	Nickname string
	Photo    string
	Email    string
	URL      string
	Rels     []string
	PGP      string
}

type Auth struct {
	Username string
	Password string
	Secret   string
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

type PostgreSQL struct {
	User     string
	Password string
	Host     string
	Database string
}
