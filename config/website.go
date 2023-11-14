package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/viper"
)

type WebsiteConfig struct {
	Title      string
	BaseURL    string
	Language   string
	Pagination int
}

func (c WebsiteConfig) Validate() error {
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

func ReadWebsiteConfig(dir string) (WebsiteConfig, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(dir)

	err := v.ReadInConfig()
	if err != nil {
		return WebsiteConfig{}, err
	}

	conf := WebsiteConfig{}
	err = v.Unmarshal(&conf)
	if err != nil {
		return WebsiteConfig{}, err
	}

	return conf, conf.Validate()
}
