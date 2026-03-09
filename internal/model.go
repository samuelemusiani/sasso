package internal

type Net struct {
	ID uint `json:"id"` // Unique identifier

	Zone string `json:"zone"` // Name of the Proxmox zone
	Name string `json:"name"` // Name of the VNet in Proxmox
	Tag  uint32 `json:"tag"`  // VXLAN tag in Proxmox

	Subnet    string `json:"subnet"`    // CIDR notation of the subnet
	Gateway   string `json:"gateway"`   // IP address of the gateway
	Broadcast string `json:"broadcast"` // Broadcast address of the subnet

	UserIDs []uint `json:"user_ids"` // IDs of users who have access to this network
}

type WireguardPeer struct {
	ID     uint `json:"id"`
	UserID uint `json:"user_id"`

	IP              string   `json:"ip"`
	PeerPrivateKey  string   `json:"peer_private_key"`
	ServerPublicKey string   `json:"server_public_key"`
	Endpoint        string   `json:"endpoint"`
	AllowedIPs      []string `json:"allowed_ips"`
}

type PortForward struct {
	ID       uint   `json:"id"`
	OutPort  uint16 `json:"out_port"`
	DestPort uint16 `json:"dest_port"`
	DestIP   string `json:"dest_ip"`
}
