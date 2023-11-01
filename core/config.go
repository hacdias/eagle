package core

import (
	"errors"
	"fmt"
	urlpkg "net/url"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Development     bool
	SourceDirectory string
	PublicDirectory string
	DataDirectory   string
	Port            int
	BaseURL         string
	TokensSecret    string
	WebhookSecret   string
	Logging         bool
	Language        string
	Title           string
	Pagination      int

	MeiliSearch   *EndpointWithKey
	User          User
	Notifications Notifications
	BunnyCDN      *BunnyCDN
	Miniflux      *Miniflux
	Linkding      *Linkding
	ImgProxy      *ImgProxy
}

func (c *Config) validate() error {
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
		return fmt.Errorf("port should be above zero")
	}

	baseUrl, err := urlpkg.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	baseUrl.Path = ""

	if baseUrl.String() != c.BaseURL {
		return fmt.Errorf("base url should be %s", baseUrl.String())
	}

	if c.Pagination < 1 {
		return errors.New("paginate must be larger than 1")
	}

	err = c.User.validate()
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) ID() string {
	return c.BaseURL + "/"
}

func (c *Config) resolvedURL(path string) *urlpkg.URL {
	url, _ := urlpkg.Parse(path)
	base, _ := urlpkg.Parse(c.BaseURL)
	return base.ResolveReference(url)
}

func (c *Config) AbsoluteURL(path string) string {
	resolved := c.resolvedURL(path)
	if resolved == nil {
		return ""
	}
	return resolved.String()
}

func (c *Config) RelativeURL(path string) string {
	resolved := c.resolvedURL(path)
	if resolved == nil {
		return path
	}

	// Take out everything before the path.
	resolved.User = nil
	resolved.Host = ""
	resolved.Scheme = ""
	return resolved.String()
}

type User struct {
	Name     string
	Username string
	Password string
	Email    string
	Photo    string
}

func (u *User) validate() error {
	if u.Username == "" {
		return errors.New("user.username is empty")
	}

	if u.Password == "" {
		return errors.New("user.password is empty")
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

type Miniflux = EndpointWithKey

type Linkding = EndpointWithKey

type ImgProxy struct {
	Directory string
	Endpoint  string
}

// ParseConfig parses the configuration from the default files and paths.
func ParseConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	err = conf.validate()
	if err != nil {
		return nil, err
	}

	return conf, nil
}
