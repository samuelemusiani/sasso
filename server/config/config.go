package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server Server `toml:"server"`
}

type Server struct {
	Bind string
}

var config Config = Config{
	Server: Server{
		Bind: ":8080",
	},
}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
