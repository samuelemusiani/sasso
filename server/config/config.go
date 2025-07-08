package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   Server   `toml:"server"`
	Database Database `toml:"database"`
	Secrets  Secrets  `toml:"secrets"`
}

type Server struct {
	Bind string
}

type Database struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Host     string `toml:"address"`
	Port     uint16 `toml:"port"`
}

type Secrets struct {
	Key  string `toml:"key"`
	Path string `toml:"path"`
}

var config Config = Config{
	Server: Server{
		Bind: ":8080",
	},
	Database: Database{
		User:     "user",
		Password: "password",
		Host:     "localhost",
		Port:     5432,
	},
	Secrets: Secrets{
		Key:  "",
		Path: "./secrets.key",
	},
}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
