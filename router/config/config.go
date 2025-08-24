package config

import "github.com/BurntSushi/toml"

type Config struct {
	Server   Server   `toml:"server"`
	Api      Api      `toml:"api"`
	Database Database `toml:"database"`
}

type Server struct {
	Bind string `toml:"bind"`
}

type Api struct {
	Secret string `toml:"secret"`
}

type Database struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	Host     string `toml:"host"`
	Port     uint16 `toml:"port"`
}

var config Config = Config{}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
