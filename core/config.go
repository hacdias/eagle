package core

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ServerConfig
	Site SiteConfig
}

// ParseConfig parses the configuration from the default files and paths.
func ParseConfig() (*Config, error) {
	serverConfig, err := parseServerConfig()
	if err != nil {
		return nil, err
	}

	siteConfig, err := parseSiteConfig(serverConfig.SourceDirectory)
	if err != nil {
		return nil, err
	}

	return &Config{
		ServerConfig: *serverConfig,
		Site:         *siteConfig,
	}, nil
}

type ServerConfig struct {
	Development     bool
	SourceDirectory string
	PublicDirectory string
	DataDirectory   string
	Port            int
	BaseURL         string // NOTE: maybe use the one from [SiteConfig].
	TokensSecret    string
	WebhookSecret   string

	Login         Login
	MeiliSearch   *EndpointWithKey
	Notifications Notifications
	BunnyCDN      *BunnyCDN
	Miniflux      *Miniflux
	Linkding      *Linkding
	ImgProxy      *ImgProxy
}

func parseServerConfig() (*ServerConfig, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")

	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &ServerConfig{}
	err = v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	err = conf.validate()
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (c *ServerConfig) validate() error {
	var err error

	c.SourceDirectory, err = filepath.Abs(c.SourceDirectory)
	if err != nil {
		return err
	}

	c.PublicDirectory, err = filepath.Abs(c.PublicDirectory)
	if err != nil {
		return err
	}

	c.DataDirectory, err = filepath.Abs(c.DataDirectory)
	if err != nil {
		return err
	}

	if c.Port < 0 {
		return fmt.Errorf("config: Port should be positive number or 0")
	}

	baseUrl, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	baseUrl.Path = ""

	if baseUrl.String() != c.BaseURL {
		return fmt.Errorf("config: BaseURL should be %s", baseUrl.String())
	}

	err = c.Login.validate()
	if err != nil {
		return err
	}

	return nil
}

func (c *ServerConfig) ID() string {
	return c.BaseURL + "/"
}

func (c *ServerConfig) resolvedURL(refStr string) *url.URL {
	ref, _ := url.Parse(refStr)
	base, _ := url.Parse(c.BaseURL)
	return base.ResolveReference(ref)
}

func (c *ServerConfig) AbsoluteURL(refStr string) string {
	resolved := c.resolvedURL(refStr)
	if resolved == nil {
		return ""
	}
	return resolved.String()
}

type Login struct {
	Username string
	Password string
}

func (u *Login) validate() error {
	if u.Username == "" {
		return errors.New("config: Login.Username is empty")
	}

	if u.Password == "" {
		return errors.New("config: Login.Password is empty")
	}

	return nil
}

type Telegram struct {
	Token  string
	ChatID int64
}

type Notifications struct {
	Telegram *Telegram
}

type BunnyCDN struct {
	Zone string
	Key  string
	Base string
}

type EndpointWithKey struct {
	Endpoint string
	Key      string
}

type Miniflux struct {
	Endpoint string
	Key      string
	JSON     string
	OPML     string
}

type Linkding struct {
	Endpoint string
	Key      string
	JSON     string
}

type ImgProxy struct {
	Directory string
	Endpoint  string
}

type SiteConfig struct {
	Paginate int
	Params   struct {
		Author struct {
			Name   string
			Email  string
			Photo  string
			Handle string
		}
	}
}

func parseSiteConfig(dir string) (*SiteConfig, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(dir)

	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &SiteConfig{}
	err = v.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	err = conf.validate()
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (c *SiteConfig) validate() error {
	if c.Paginate < 1 {
		return errors.New("hugo config: .Paginate must be larger than 1")
	}

	return nil
}
