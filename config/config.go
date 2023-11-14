package config

type Config struct {
	Server  ServerConfig
	Website WebsiteConfig
}

func ReadConfig(dir string) (*Config, error) {
	server, err := ReadServerConfig(dir)
	if err != nil {
		return nil, err
	}

	website, err := ReadWebsiteConfig(server.Source)
	if err != nil {
		return nil, err
	}

	return &Config{
		Server:  server,
		Website: website,
	}, nil
}
