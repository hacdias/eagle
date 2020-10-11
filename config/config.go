package config

import (
	"github.com/spf13/viper"
)

func Get() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("port", 8080)
	viper.SetDefault("domain", "http://localhost:8080")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	conf.setupLogger()
	return conf, nil
}
