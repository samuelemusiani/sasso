package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   Server   `toml:"server"`
	Database Database `toml:"database"`
	// Secrets   Secrets   `toml:"secrets"`
	Wireguard       Wireguard       `toml:"wireguard"`
	Firewall        Firewall        `toml:"firewall"`
	WBInterfaceName WBInterfaceName `toml:"wb_interface_name"`
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

type Firewall struct {
	VPNZone   string `toml:"vpn"`
	SassoZone string `toml:"sasso"`
}

type WBInterfaceName struct {
	InterfaceName string `toml:"sasso"`
}

var config Config = Config{}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
