package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port   int
	Source string
}

func (c ServerConfig) Validate() error {
	if _, err := os.Stat(c.Source); err != nil {
		return fmt.Errorf("server config: Source %q does not exist: %w", c.Source, err)
	}

	if c.Port < 0 {
		return fmt.Errorf("server config: Port must be above 0")
	}

	return nil
}

func ReadServerConfig(dir string) (ServerConfig, error) {
	v := viper.New()
	v.SetConfigName("eagle")
	v.AddConfigPath(dir)

	err := v.ReadInConfig()
	if err != nil {
		return ServerConfig{}, err
	}

	conf := ServerConfig{}
	err = v.Unmarshal(&conf)
	if err != nil {
		return ServerConfig{}, err
	}

	return conf, conf.Validate()
}
