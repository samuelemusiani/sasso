package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server    Server    `toml:"server"`
	Database  Database  `toml:"database"`
	Wireguard Wireguard `toml:"wireguard"`
	Firewall  Firewall  `toml:"firewall"`
}

type Server struct {
	Endpoint string `toml:"endpoint"`
	Secret   string `toml:"secret"`
}

type Database struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	Host     string `toml:"host"`
	Port     uint16 `toml:"port"`
}

type Wireguard struct {
	PublicKey string `toml:"public_key"`
	Endpoint  string `toml:"endpoint"`
	VPNSubnet string `toml:"vpn_subnet"`
	VMsSubnet string `toml:"vms_subnet"`
	Interface string `toml:"interface_name"`
}

type Firewall struct {
	VPNZone   string `toml:"vpn"`
	SassoZone string `toml:"sasso"`
}

var config Config = Config{}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)

	return err
}
