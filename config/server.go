package config

import "github.com/spf13/viper"

type ServerConfig struct {
	Port   int
	Source string
}

func (c ServerConfig) Validate() error {
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
