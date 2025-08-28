package ticket

import (
	"samuelemusiani/sasso/router/db"
	"samuelemusiani/sasso/router/fw"
	"samuelemusiani/sasso/router/gateway"
	"samuelemusiani/sasso/router/utils"
)

type RequestType string

var (
	TypeNewNetworkRequest    RequestType = "new-network"
	TypeDeleteNetworkRequest RequestType = "delete-network"
)

type Request interface {
	GetType() RequestType
	Execute(gateway.Gateway) error  // Execute the request, performing the necessary actions
	SaveToDB(ticketID string) error // Save the request to the database. Ticket ID is used to link the request to a ticket
}

func requestFromDBByTicket(t *db.Ticket) (Request, error) {
	switch t.RequestType {
	case string(TypeNewNetworkRequest):
		r, err := db.GetNetworkRequestByTicket(t)
		if err != nil {
			return nil, err
		}
		return &NetworkRequest{
			VNet:      r.VNet,
			VNetID:    r.VNetID,
			Status:    r.Status,
			Success:   r.Success,
			Error:     r.Error,
			Subnet:    r.Subnet,
			RouterIP:  r.RouterIP,
			Broadcast: r.Broadcast,
		}, nil
	case string(TypeDeleteNetworkRequest):
		r, err := db.GetDeleteNetworkRequestByTicket(t)
		if err != nil {
			return nil, err
		}
		return &DeleteNetworkRequest{
			VNet:    r.VNet,
			VNetID:  r.VNetID,
			Status:  r.Status,
			Success: r.Success,
			Error:   r.Error,
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

func (nr *NetworkRequest) GetType() RequestType {
	return TypeNewNetworkRequest
}

func (nr *NetworkRequest) Execute(gtw gateway.Gateway) error {
	s, err := utils.NextAvailableSubnet()
	if err != nil {
		logger.With("error", err).Error("Failed to get next available subnet")
		nr.Success = false
		nr.Error = err.Error()
		nr.Status = "failed"
		return err
	}
	nr.Subnet = s

	gt, err := utils.GatewayAddressFromSubnet(s)
	if err != nil {
		logger.With("error", err).Error("Failed to get gateway address from subnet")
		nr.Success = false
		nr.Error = err.Error()
		nr.Status = "failed"
		return err
	}
	nr.RouterIP = gt

	br, err := utils.GetBroadcastAddressFromSubnet(s)
	if err != nil {
		logger.With("error", err).Error("Failed to get broadcast address from subnet")
		nr.Success = false
		nr.Error = err.Error()
		nr.Status = "failed"
		return err
	}
	nr.Broadcast = br

	inter, err := gtw.NewInterface(nr.VNet, nr.VNetID, nr.Subnet, nr.RouterIP, nr.Broadcast)
	if err != nil {
		logger.With("error", err).Error("Failed to create new interface on gateway")
		nr.Success = false
		nr.Error = err.Error()
		nr.Status = "failed"
		return err
	}

	err = inter.SaveToDB()
	if err != nil {
		logger.With("error", err).Error("Failed to save interface to database")
		nr.Success = false
		nr.Error = err.Error()
		nr.Status = "failed"
		return err
	}

	err = fw.NewInterface(inter)
	if err != nil {
		logger.With("error", err).Error("Failed to create new interface on firewall")
		nr.Success = false
		nr.Error = err.Error()
		nr.Status = "failed"
		return err
	}

	nr.Status = "completed"
	nr.Success = true
	return nil
}

func NetworkRequestFromDB(r *db.NetworkRequest) NetworkRequest {
	return NetworkRequest{
		VNet:      r.VNet,
		VNetID:    r.VNetID,
		Status:    r.Status,
		Success:   r.Success,
		Error:     r.Error,
		Subnet:    r.Subnet,
		RouterIP:  r.RouterIP,
		Broadcast: r.Broadcast,
	}
}

func NewNetworkRequest(vnet string, vnetID uint) NetworkRequest {
	return NetworkRequest{
		VNet:   vnet,
		VNetID: vnetID,
		Status: "pending",
	}
}

func (nr *NetworkRequest) SaveToDB(ticketID string) error {
	return db.SaveNetworkRequest(db.NetworkRequest{
		Ticket: db.Ticket{
			UUID:        ticketID,
			RequestType: string(nr.GetType()),
		},
		VNet:      nr.VNet,
		VNetID:    nr.VNetID,
		Status:    nr.Status,
		Success:   nr.Success,
		Error:     nr.Error,
		Subnet:    nr.Subnet,
		RouterIP:  nr.RouterIP,
		Broadcast: nr.Broadcast,
	})
}

type DeleteNetworkRequest struct {
	VNet   string `json:"vnet"`    // Name of the VNet to delete
	VNetID uint   `json:"vnet_id"` // ID of the VNet to delete

	Status  string // Status of the request
	Success bool   // True if the request was successful
	Error   string // Error message if the request failed
}

func (dr *DeleteNetworkRequest) GetType() RequestType {
	return TypeDeleteNetworkRequest
}

func (dr *DeleteNetworkRequest) Execute(gtw gateway.Gateway) error {
	var dbIface *db.Interface
	var err error

	if dr.VNet != "" {
		dbIface, err = db.GetInterfaceByVNet(dr.VNet)
	} else {
		dbIface, err = db.GetInterfaceByVNetID(dr.VNetID)
	}

	iface := gateway.InterfaceFromDB(dbIface)

	if err != nil {
		logger.With("error", err, "vnet", dr.VNet, "vnet_id", dr.VNetID).Error("Failed to get interface from database")
		dr.Success = false
		dr.Error = err.Error()
		dr.Status = "failed"
		return err
	}

	err = gtw.RemoveInterface(iface.LocalID)
	if err != nil {
		logger.With("error", err, "local_id", iface.LocalID).Error("Failed to remove interface from gateway") dr.Success = false
		dr.Error = err.Error()
		dr.Status = "failed"
		return err
	}

	err = db.DeleteInterface(iface.ID)
	if err != nil {
		logger.With("error", err, "interface_id", iface.ID).Error("Failed to delete interface from database")
		dr.Success = false
		dr.Error = err.Error()
		dr.Status = "failed"
		return err
	}

	err = fw.DeleteInterface(iface)
	if err != nil {
		logger.With("error", err, "interface_id", iface.ID).Error("Failed to delete interface from firewall")
		dr.Success = false
		dr.Error = err.Error()
		dr.Status = "failed"
		return err
	}

	dr.Status = "completed"
	dr.Success = true
	return nil
}

func (dr *DeleteNetworkRequest) SaveToDB(ticketID string) error {
	return db.SaveDeleteNetworkRequest(db.DeleteNetworkRequest{
		Ticket: db.Ticket{
			UUID:        ticketID,
			RequestType: string(dr.GetType()),
		},
		VNet:    dr.VNet,
		VNetID:  dr.VNetID,
		Status:  dr.Status,
		Success: dr.Success,
		Error:   dr.Error,
	})
}

func NewDeleteNetworkRequest(vnet string, vnetID uint) DeleteNetworkRequest {
	return DeleteNetworkRequest{
		VNet:   vnet,
		VNetID: vnetID,
		Status: "pending",
	}
}
