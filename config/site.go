package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/viper"
)

type SiteConfig struct {
	Title      string
	BaseURL    string
	Language   string
	Pagination int
	Assets     []Asset
}

type Asset struct {
	Name  string
	Files []string
}

func (c SiteConfig) Validate() error {
	baseUrl, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	baseUrl.Path = ""

	if baseUrl.String() != c.BaseURL {
		return fmt.Errorf("website config: BaseURL should be %s", baseUrl.String())
	}

	if c.Pagination < 1 {
		return errors.New("website config: Pagination must be larger than 1")
	}

	return nil
}

func ReadWebsiteConfig(dir string) (SiteConfig, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(dir)

	err := v.ReadInConfig()
	if err != nil {
		return SiteConfig{}, err
	}

	conf := SiteConfig{}
	err = v.Unmarshal(&conf)
	if err != nil {
		return SiteConfig{}, err
	}

	return conf, conf.Validate()
}
