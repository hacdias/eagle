package config

import "go.uber.org/zap"

type Config struct {
	logger      *zap.Logger `mapstructure:"-"`
	Port        int
	Development bool
	Domain      string // MUST NOT CONTAIN END SLASH
	Source      string
	Site        Site

	// OLD
	Hugo Hugo
}

func (c *Config) S() *zap.SugaredLogger {
	return c.logger.Sugar()
}

func (c *Config) L() *zap.Logger {
	return c.logger
}

type Site struct {
	Domain      string
	Title       string
	Menu        []MenuItem
	Author      Author
	Description string
}

type Author struct {
	Username string
	Name     string
	Avatar   string
	Cover    string
	Homepage string
}

type MenuItem struct {
	Name string
	URL  string
}

type Hugo struct {
	Source      string
	Destination string
}
