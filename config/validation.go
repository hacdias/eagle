package config

import (
	"errors"
	"fmt"
	"path/filepath"

	urlpkg "net/url"

	"github.com/hacdias/eagle/v4/entry/mf2"
	"github.com/thoas/go-funk"
)

func (t *Tor) validate() error {
	if t.Directory == "" {
		return fmt.Errorf("tor.directory must be set")
	}

	return nil
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

	return nil
}

func (s *Source) validate() error {
	var err error
	s.Directory, err = filepath.Abs(s.Directory)
	if err != nil {
		return err
	}

	// TODO: validate assets?
	return nil
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

	return nil
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
