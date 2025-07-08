package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   Server   `toml:"server"`
	Database Database `toml:"database"`
	Secrets  Secrets  `toml:"secrets"`
	Proxmox  Proxmox  `toml:"proxmox"`
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

type Proxmox struct {
	Url                string `toml:"url"`
	TokenID            string `toml:"token_id"`
	Secret             string `toml:"secret"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"`
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
	Proxmox: Proxmox{
		Url:                "https://proxmox.local:8006",
		InsecureSkipVerify: true,
		TokenID:            "root@pam!sasso",
		Secret:             "super-iper-secret-token",
	},
}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
