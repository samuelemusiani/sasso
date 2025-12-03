package config

import "github.com/BurntSushi/toml"

type Config struct {
	Server   Server   `toml:"server"`
	Database Database `toml:"database"` // DONE
	Network  Network  `toml:"network"`  // DONE
	Gateway  Gateway  `toml:"gateway"`  // DONE
	Firewall Firewall `toml:"firewall"` // DONE
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
	Linux   LinuxGatewayConfig   `toml:"linux"`
}

type ProxmoxGatewayConfig struct {
	Url                string `toml:"url"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"`
	TokenID            string `toml:"token_id"`
	Secret             string `toml:"secret"`
	VMID               uint   `toml:"vmid"`
}

type LinuxGatewayConfig struct {
	Port  uint16   `toml:"port"`
	Peers []string `toml:"peers"`
	MTU   uint16   `toml:"mtu"`
}

type Database struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	Host     string `toml:"host"`
	Port     uint16 `toml:"port"`
}

type Firewall struct {
	Type      string                  `toml:"type"`
	Shorewall ShorewallFirewallConfig `toml:"shorewall"`
}

type ShorewallFirewallConfig struct {
	ExternalZone string `toml:"external_zone"`
	VMZone       string `toml:"vm_zone"`
	PublicIP     string `toml:"public_ip"`
}

var config Config = Config{}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
