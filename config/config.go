package config

import (
	urlpkg "net/url"

	"github.com/hacdias/eagle/v4/pkg/mf2"
	"github.com/spf13/viper"
)

type Tor struct {
	Directory string
	Logging   bool
}

type Server struct {
	Port          int
	BaseURL       string
	TokensSecret  string
	WebhookSecret string
	TilesSource   string
	Tor           *Tor
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

type Asset struct {
	Name  string
	Files []string
}

type Source struct {
	Directory string
	Assets    []Asset
}

type PostgreSQL struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
}

type MenuItem struct {
	Name string
	Link string
}

type Site struct {
	Language     string
	Title        string
	Description  string
	Pagination   int
	ChromaTheme  string
	Sections     []string
	IndexSection string
	Menus        map[string][]MenuItem
}

type User struct {
	Name       string
	Username   string
	Password   string
	Email      string
	Photo      string
	Identities []string
}

type Syndication struct {
	Twitter bool
	Reddit  bool
}

type Telegram struct {
	Token  string
	ChatID int64
}

type Notifications struct {
	Telegram *Telegram
}

type PostType struct {
	Type       string   `json:"type"`
	Name       string   `json:"name"`
	Properties []string `json:"properties,omitempty"`
	Required   []string `json:"required-properties,omitempty"`
}

type Micropub struct {
	Sections  map[mf2.Type][]string
	PostTypes []PostType
}

type Webmentions struct {
	Secret         string
	DisableSending bool
}

type XRay struct {
	Endpoint string
	Twitter  bool
	Reddit   bool
}

type Twitter struct {
	User        string
	Key         string
	Secret      string
	Token       string
	TokenSecret string
}

type Reddit struct {
	User     string
	Password string
	App      string
	Secret   string
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

type Lastfm struct {
	Key  string
	User string
}

type ImgProxy struct {
	Directory string
	Endpoint  string
}

type Config struct {
	Development   bool
	Server        Server
	Source        Source
	PostgreSQL    PostgreSQL
	Site          Site
	User          User
	Syndications  Syndication
	Notifications Notifications
	Micropub      Micropub
	Webmentions   Webmentions
	XRay          *XRay
	Twitter       *Twitter
	Reddit        *Reddit
	BunnyCDN      *BunnyCDN
	Miniflux      *Miniflux
	Lastfm        *Lastfm
	ImgProxy      *ImgProxy
}

func (c *Config) ID() string {
	return c.Server.BaseURL + "/"
}

// Parse parses the configuration from the default files and paths.
func Parse() (*Config, error) {
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
