package eagle

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

	Server        Server
	PostgreSQL    PostgreSQL
	Site          Site
	User          User
	Notifications Notifications
	Webmentions   Webmentions
	XRay          *XRay
	BunnyCDN      *BunnyCDN
	Miniflux      *Miniflux
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

	err = c.Server.validate()
	if err != nil {
		return err
	}

	err = c.PostgreSQL.validate()
	if err != nil {
		return err
	}

	err = c.Site.validate()
	if err != nil {
		return err
	}

	err = c.User.validate()
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) ID() string {
	return c.Server.BaseURL + "/"
}

type Server struct {
	Port          int
	BaseURL       string
	TokensSecret  string
	WebhookSecret string
	Logging       bool
}

func (s *Server) validate() error {
	if s.Port < 0 {
		return fmt.Errorf("port should be above zero")
	}

	baseUrl, err := urlpkg.Parse(s.BaseURL)
	if err != nil {
		return err
	}
	baseUrl.Path = ""

	if baseUrl.String() != s.BaseURL {
		return fmt.Errorf("base url should be %s", baseUrl.String())
	}

	return nil
}

func (s *Server) resolvedURL(path string) *urlpkg.URL {
	url, _ := urlpkg.Parse(path)
	base, _ := urlpkg.Parse(s.BaseURL)
	return base.ResolveReference(url)
}

func (s *Server) AbsoluteURL(path string) string {
	resolved := s.resolvedURL(path)
	if resolved == nil {
		return ""
	}
	return resolved.String()
}

func (s *Server) RelativeURL(path string) string {
	resolved := s.resolvedURL(path)
	if resolved == nil {
		return path
	}

	// Take out everything before the path.
	resolved.User = nil
	resolved.Host = ""
	resolved.Scheme = ""
	return resolved.String()
}

type PostgreSQL struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
}

func (p *PostgreSQL) validate() error {
	if p.User == "" {
		return errors.New("postgresql.user is missing")
	}

	if p.Password == "" {
		return errors.New("postgresql.password is missing")
	}

	if p.Host == "" {
		return errors.New("postgresql.host is missing")
	}

	if p.Database == "" {
		return errors.New("postgresql.database is missing")
	}

	if p.Port == "" {
		return errors.New("postgresql.port is missing")
	}

	return nil
}

type Site struct {
	Language    string
	Title       string
	Description string
	Pagination  int
}

func (s *Site) validate() error {
	if s.Pagination < 1 {
		return errors.New("paginate must be larger than 1")
	}

	return nil
}

type User struct {
	Name       string
	Username   string
	Password   string
	Email      string
	Photo      string
	Identities []string
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

type Webmentions struct {
	Secret         string
	DisableSending bool
}

type XRay struct {
	Endpoint string
}

type BunnyCDN struct {
	Zone string
	Key  string
	Base string
}

type Miniflux struct {
	Endpoint string
	Key      string
}

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
