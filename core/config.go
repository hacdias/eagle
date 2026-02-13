package core

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ServerConfig
	Site SiteConfig
}

// ParseConfig parses the configuration from the default files and paths.
func ParseConfig(baseURL string) (*Config, error) {
	serverConfig, err := parseServerConfig()
	if err != nil {
		return nil, err
	}

	siteConfig, err := parseSiteConfig(serverConfig.SourceDirectory, baseURL)
	if err != nil {
		return nil, err
	}

	return &Config{
		ServerConfig: *serverConfig,
		Site:         *siteConfig,
	}, nil
}

func (c *Config) ID() string {
	return c.Site.BaseURL + "/"
}

func (c *Config) resolvedURL(refStr string) *url.URL {
	ref, _ := url.Parse(refStr)
	base, _ := url.Parse(c.Site.BaseURL)
	return base.ResolveReference(ref)
}

func (c *Config) AbsoluteURL(refStr string) string {
	resolved := c.resolvedURL(refStr)
	if resolved == nil {
		return ""
	}
	return resolved.String()
}

type ServerConfig struct {
	Development     bool
	SourceDirectory string
	PublicDirectory string
	DataDirectory   string
	Port            int
	TokensSecret    string
	WebhookSecret   string
	Tor             bool

	Login         Login
	Comments      Comments
	Webmentions   Webmentions
	Notifications Notifications
	Media         Media
	Meilisearch   *Meilisearch
	Plugins       map[string]map[string]any
}

func parseServerConfig() (*ServerConfig, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")

	v.SetEnvPrefix("eagle")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

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

	err = c.Login.validate()
	if err != nil {
		return err
	}

	err = c.Comments.validate()
	if err != nil {
		return err
	}

	err = c.Media.validate()
	if err != nil {
		return err
	}

	return nil
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

type Comments struct {
	Redirect string
	Captcha  string
}

func (c *Comments) validate() error {
	c.Captcha = strings.ToLower(c.Captcha)
	return nil
}

type Webmentions struct {
	Secret string
}

type Telegram struct {
	Token  string
	ChatID int64
}

type Notifications struct {
	Telegram *Telegram
}

type FileSystem struct {
	Base      string
	Directory string
}

type Bunny struct {
	Zone string
	Key  string
	Base string
}

type ImgProxy struct {
	Directory string
	Endpoint  string
}

type Media struct {
	Storage struct {
		Bunny      *Bunny
		FileSystem *FileSystem
	}

	Transformer struct {
		ImgProxy *ImgProxy
	}
}

func (m *Media) validate() error {
	if m.Storage.Bunny != nil && m.Storage.FileSystem != nil {
		return errors.New("config: Media.Storage can only have one of Bunny or FileSystem")
	}

	if m.Storage.Bunny == nil && m.Storage.FileSystem == nil {
		return errors.New("config: Media.Storage must have one of Bunny or FileSystem")
	}

	if m.Transformer.ImgProxy == nil {
		return errors.New("config: Media.Transformer.ImgProxy is required")
	}

	return nil
}

type Meilisearch struct {
	Endpoint string
	Key      string
}

type SiteConfig struct {
	BaseURL      string
	Title        string
	LanguageCode string
	Taxonomies   map[string]string
	Pagination   struct {
		PagerSize int
	}
	Params struct {
		Author struct {
			Name   string
			Email  string
			Photo  string
			Handle string
		}
		Site struct {
			Description string
		}
	}
}

func parseSiteConfig(dir string, baseURL string) (*SiteConfig, error) {
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

	if baseURL != "" {
		conf.BaseURL = baseURL
	}

	err = conf.validate()
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (c *SiteConfig) validate() error {
	if c.Pagination.PagerSize < 1 {
		return errors.New("hugo config: .Pagination.PagerSize must be larger than 1")
	}

	baseUrl, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	baseUrl.Path = ""

	if baseUrl.String() != strings.TrimSuffix(c.BaseURL, "/") {
		return fmt.Errorf("config: BaseURL should be %s", baseUrl.String())
	}
	// BaseURL is always without trailing slash.
	c.BaseURL = baseUrl.String()

	return nil
}
