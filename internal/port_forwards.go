package internal

type PortForward struct {
	ID       uint   `json:"id"`
	OutPort  uint16 `json:"out_port"`
	DestPort uint16 `json:"dest_port"`
	DestIP   string `json:"dest_ip"`
	UserID   uint   `json:"user_id"`
	Approved bool   `json:"approved"`
}
