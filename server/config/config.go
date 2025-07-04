package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   Server   `toml:"server"`
	Database Database `toml:"database"`
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
}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
