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
	Bind string `toml:"bind"`
}

type Database struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Host     string `toml:"host"`
	Port     uint16 `toml:"port"`
}

type Secrets struct {
	Key  string `toml:"key"`
	Path string `toml:"path"`

	InternalSecret string `toml:"internal_secret"`
}

type Proxmox struct {
	Url                string          `toml:"url"`
	TokenID            string          `toml:"token_id"`
	Secret             string          `toml:"secret"`
	InsecureSkipVerify bool            `toml:"insecure_skip_verify"`
	Template           ProxmoxTemplate `toml:"template"`
	Clone              ProxmoxClone    `toml:"clone"`
	Network            ProxmoxNetwork  `toml:"network"`
	Backup             ProxmoxBackup   `toml:"backup"`
}

type ProxmoxTemplate struct {
	Node string `toml:"node"`
	VMID int    `toml:"vmid"`
}

type ProxmoxClone struct {
	TargetNode     string `toml:"target_node"`
	IDTemplate     string `toml:"id_template"`
	VMIDUserDigits int    `toml:"vmid_user_digits"`
	VMIDVMDigits   int    `toml:"vmid_vm_digits"`
	Full           bool   `toml:"full"`
}

type ProxmoxNetwork struct {
	SDNZone      string `toml:"sdn_zone"`
	VXLANIDStart uint32 `toml:"vxlan_id_start"`
	VXLANIDEnd   uint32 `toml:"vxlan_id_end"`
}

type ProxmoxBackup struct {
	Storage string `toml:"storage"`
}

var config Config = Config{}

func Get() *Config {
	return &config
}

func Parse(path string) error {
	_, err := toml.DecodeFile(path, &config)
	return err
}
