package internal

type Net struct {
	ID uint `json:"id"` // Unique identifier

	Zone string `json:"zone"` // Name of the Proxmox zone
	Name string `json:"name"` // Name of the VNet in Proxmox
	Tag  uint32 `json:"tag"`  // VXLAN tag in Proxmox

	Subnet    string `gorm:"not null"` // CIDR notation of the subnet
	Gateway   string `gorm:"not null"` // IP address of the gateway
	Broadcast string `gorm:"not null"` // Broadcast address of the subnet

	UserID uint `json:"user_id"` // ID of the user who owns this net
}

type VPNUpdate struct {
	UserID    uint   `json:"user_id"`
	VPNConfig string `json:"vpn_config"`
}
