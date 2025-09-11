package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   Server   `toml:"server"`
	Database Database `toml:"database"`
	// Secrets   Secrets   `toml:"secrets"`
	Wireguard Wireguard `toml:"wireguard"`
}

type Server struct {
	Bind string `toml:"bind"`
}

type Database struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Host     string `toml:"host"`
	Port     uint16 `toml:"port"`
}

type Wireguard struct {
	PublicKey string `toml:"public_key"`
	Endpoint  string `toml:"endpoint"`
	Subnet    string `toml:"subnet"`
}

var config Config = Config{}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
