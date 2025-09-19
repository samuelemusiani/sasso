package config

import "github.com/BurntSushi/toml"

type Config struct {
	Server   Server   `toml:"server"`
	Database Database `toml:"database"`
	Network  Network  `toml:"network"`
	Gateway  Gateway  `toml:"gateway"`
}

type Server struct {
	Endpoint string `toml:"endpoint"`
	Secret   string `toml:"secret"`
}

type Network struct {
	UsableSubnet    string `toml:"usable_subnet"`
	NewSubnetPrefix int    `toml:"new_subnet_prefix"`
}

type Gateway struct {
	Type    string               `toml:"type"`
	Proxmox ProxmoxGatewayConfig `toml:"proxmox"`
}

type ProxmoxGatewayConfig struct {
	Url                string `toml:"url"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"`
	TokenID            string `toml:"token_id"`
	Secret             string `toml:"secret"`
	VMID               uint   `toml:"vmid"`
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
