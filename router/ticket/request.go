package ticket

import "samuelemusiani/sasso/router/db"

type RequestType string

var (
	TypeNewNetworkRequest RequestType = "new-network"
)

type Request interface {
	GetType() RequestType
	Execute() error                 // Execute the request, performing the necessary actions
	SaveToDB(ticketID string) error // Save the request to the database. Ticket ID is used to link the request to a ticket
}

func requestFromDBByTicket(t *db.Ticket) (Request, error) {
	switch t.RequestType {
	case string(TypeNewNetworkRequest):
		r, err := db.GetNetworkRequestByTicket(t)
		if err != nil {
			return nil, err
		}
		return NetworkRequest{
			VNet:      r.VNet,
			VNetID:    r.VNetID,
			Status:    r.Status,
			Success:   r.Success,
			Error:     r.Error,
			Subnet:    r.Subnet,
			RouterIP:  r.RouterIP,
			Broadcast: r.Broadcast,
		}, nil
	default:
		return nil, db.ErrNotFound
	}
}

type NetworkRequest struct {
	VNet   string // Name of the new VNet
	VNetID uint   // ID of the new VNet (VXLAN ID)

	Status  string // Status of the request
	Success bool   // True if the request was successful
	Error   string // Error message if the request failed

	Subnet    string // Subnet of the new VNet
	RouterIP  string // Router IP of the new VNet
	Broadcast string // Broadcast address of the new VNet
}

func (nr NetworkRequest) GetType() RequestType {
	return TypeNewNetworkRequest
}

func (nr NetworkRequest) Execute() error {
	// Implement the logic to execute the network request
	// This could involve creating a new network, configuring it, etc.
	// For now, we'll just return nil to indicate success
	return nil
}

func NewNetworkRequest(vnet string, vnetID uint) NetworkRequest {
	return NetworkRequest{
		VNet:   vnet,
		VNetID: vnetID,
		Status: "pending",
	}
}

func (nr NetworkRequest) SaveToDB(ticketID string) error {
	return db.SaveNetworkRequest(db.NetworkRequest{
		Ticket: db.Ticket{
			UUID:        ticketID,
			RequestType: string(nr.GetType()),
		},
		VNet:   nr.VNet,
		VNetID: nr.VNetID,
		Status: nr.Status,
	})
}
