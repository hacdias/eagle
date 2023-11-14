package config

import "github.com/spf13/viper"

type WebsiteConfig struct {
	Title      string
	BaseURL    string
	Language   string
	Pagination int
}

func (c WebsiteConfig) Validate() error {
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
