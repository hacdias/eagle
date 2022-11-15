package eagle

import (
	"errors"
	"fmt"
	urlpkg "net/url"
	"path/filepath"

	"github.com/hacdias/eagle/pkg/mf2"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
)

type ActivityPub struct {
	Directory string
}

func (a *ActivityPub) validate() error {
	if a.Directory == "" {
		return fmt.Errorf("activitypub.directory must be set")
	}

	var err error
	a.Directory, err = filepath.Abs(a.Directory)
	return err
}

type Tor struct {
	Directory string
	Logging   bool
}

func (t *Tor) validate() error {
	if t.Directory == "" {
		return fmt.Errorf("tor.directory must be set")
	}

	var err error
	t.Directory, err = filepath.Abs(t.Directory)
	return err
}

type Server struct {
	Port          int
	BaseURL       string
	TokensSecret  string
	WebhookSecret string
	TilesSource   string
	Logging       bool
	ActivityPub   *ActivityPub
	Tor           *Tor
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

	if s.Tor != nil {
		if err = s.Tor.validate(); err != nil {
			return err
		}
	}

	if s.ActivityPub != nil {
		if err = s.ActivityPub.validate(); err != nil {
			return err
		}
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

type Asset struct {
	Name  string
	Files []string
}

type Source struct {
	Directory string
	Assets    []Asset
}

func (s *Source) validate() error {
	var err error
	s.Directory, err = filepath.Abs(s.Directory)
	if err != nil {
		return err
	}

	return nil
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
	Taxonomies   map[string]Taxonomy
}

func (s *Site) validate() error {
	if s.Pagination < 1 {
		return errors.New("paginate must be larger than 1")
	}

	if s.IndexSection == "" {
		return errors.New("indexSection must be configured")
	}

	if !funk.ContainsString(s.Sections, s.IndexSection) {
		return errors.New("sections must include IndexSection")
	}

	if len(s.Sections) != len(funk.UniqString(s.Sections)) {
		return errors.New("sections includes duplicate entries")
	}

	for _, taxonomy := range s.Taxonomies {
		if err := taxonomy.validate(); err != nil {
			return err
		}
	}

	return nil
}

type Taxonomy struct {
	Title    string
	Singular string
}

func (t *Taxonomy) validate() error {
	if t.Title == "" {
		return errors.New("tag must have title")
	}

	if t.Singular == "" {
		return errors.New("tag must have singular version")
	}

	return nil
}

type User struct {
	Name       string
	Username   string
	Password   string
	Email      string
	Photo      string
	CoverPhoto string
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

func (m *Micropub) validate() error {
	for mf2Type := range m.Sections {
		if !mf2.IsType(mf2Type) {
			return fmt.Errorf("%s is not a valid micropub type", mf2Type)
		}
	}

	return nil
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

func (m *Micropub) AllowedTypes() []mf2.Type {
	var allowedTypes []mf2.Type
	for typ := range m.Sections {
		allowedTypes = append(allowedTypes, typ)
	}
	return allowedTypes
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

func (c *Config) validate() error {
	err := c.Server.validate()
	if err != nil {
		return err
	}

	err = c.Source.validate()
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

	if c.Syndications.Twitter && c.Twitter == nil {
		return errors.New("syndication.twitter is true but twitter is not defined")
	}

	if c.Syndications.Reddit && c.Reddit == nil {
		return errors.New("syndication.reddit is true but reddit is not defined")
	}

	err = c.Micropub.validate()
	if err != nil {
		return err
	}

	micropubSections := []string{}
	for _, sections := range c.Micropub.Sections {
		micropubSections = append(micropubSections, sections...)
	}
	micropubSections = funk.UniqString(micropubSections)

	intersect := funk.IntersectString(c.Site.Sections, micropubSections)
	if len(intersect) != len(micropubSections) {
		return fmt.Errorf("Micropub.Sections can only use sections defined in Sections")
	}

	if c.XRay != nil {
		if c.XRay.Twitter && c.Twitter == nil {
			return errors.New("xray.twitter is true but twitter is not defined")
		}

		if c.XRay.Reddit && c.Reddit == nil {
			return errors.New("xray.reddit is true but reddit is not defined")
		}
	}

	return nil
}

func (c *Config) ID() string {
	return c.Server.BaseURL + "/"
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
