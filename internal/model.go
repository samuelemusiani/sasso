package internal

type Net struct {
	ID uint `json:"id"` // Unique identifier

	Zone string `json:"zone"` // Name of the Proxmox zone
	Name string `json:"name"` // Name of the VNet in Proxmox
	Tag  uint32 `json:"tag"`  // VXLAN tag in Proxmox

	Subnet    string `gorm:"not null"` // CIDR notation of the subnet
	Gateway   string `gorm:"not null"` // IP address of the gateway
	Broadcast string `gorm:"not null"` // Broadcast address of the subnet

	UserIDs []uint `json:"user_ids"` // IDs of users who have access to this network
}

type VPNUpdate struct {
	ID        uint   `json:"id"`
	VPNConfig string `json:"vpn_config"`
	VPNIP     string `json:"vpn_ip"`
}

type PortForward struct {
	ID       uint   `json:"id"`
	OutPort  uint16 `json:"out_port"`
	DestPort uint16 `json:"dest_port"`
	DestIP   string `json:"dest_ip"`
}
