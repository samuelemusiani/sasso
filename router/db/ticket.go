package db

import "gorm.io/gorm"

type Ticket struct {
	UUID        string `gorm:"primaryKey"`
	RequestType string `gorm:"not null"`
}

type NetworkRequest struct {
	Ticket
	VNet   string `gorm:"not null"` // Name of the new VNet
	VNetID uint   `gorm:"not null"` // ID of the new VNet (VXLAN ID)

	Status  string `gorm:"not null;default:'pending'"` // Status of the request
	Success bool   `gorm:"not null;default:false"`     // True if the request was successful
	Error   string `gorm:"not null;default:''"`        // Error message if the request failed

	Subnet    string `gorm:"not null"` // Subnet of the new VNet
	RouterIP  string `gorm:"not null"` // Router IP of the new VNet
	Broadcast string `gorm:"not null"` // Broadcast address of the new VNet
}

type DeleteNetworkRequest struct {
	Ticket
	VNet   string
	VNetID uint

	Status  string `gorm:"not null;default:'pending'"`
	Success bool   `gorm:"not null;default:false"`
	Error   string `gorm:"not null;default:''"`
}

func initTickets() error {
	return db.AutoMigrate(&Ticket{}, &NetworkRequest{}, &DeleteNetworkRequest{})
}

func GetTicketByUUID(uuid string) (*Ticket, error) {
	var t Ticket
	if err := db.First(&t, "uuid = ?", uuid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.With("error", err).Error("Failed to retrieve ticket by UUID")
		return nil, err
	}

	return &t, nil
}

func GetNetworkRequestByTicket(t *Ticket) (*NetworkRequest, error) {
	var req NetworkRequest
	if err := db.First(&req, "uuid = ?", t.UUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.With("error", err).Error("Failed to retrieve network request by ticket")
		return nil, err
	}

	return &req, nil
}

func GetPendingNetworkRequests() ([]NetworkRequest, error) {
	var reqs []NetworkRequest
	if err := db.Where("status = ?", "pending").Find(&reqs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []NetworkRequest{}, nil
		}
		logger.With("error", err).Error("Failed to retrieve pending network requests")
		return nil, err
	}
	return reqs, nil
}

func UpdateNetworkRequest(req *NetworkRequest) error {
	return db.Save(req).Error
}

func SaveNetworkRequest(req NetworkRequest) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&req.Ticket).Error; err != nil {
			logger.With("error", err).Error("Failed to create network request")
			return err
		}

		if err := tx.Save(&req).Error; err != nil {
			logger.With("error", err).Error("Failed to create network request details")
			return err
		}

		return nil
	})
}

func SaveDeleteNetworkRequest(req DeleteNetworkRequest) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&req.Ticket).Error; err != nil {
			logger.With("error", err).Error("Failed to create delete network request")
			return err
		}

		if err := tx.Save(&req).Error; err != nil {
			logger.With("error", err).Error("Failed to create delete network request details")
			return err
		}

		return nil
	})
}

func GetDeleteNetworkRequestByTicket(t *Ticket) (*DeleteNetworkRequest, error) {
	var req DeleteNetworkRequest
	if err := db.First(&req, "uuid = ?", t.UUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		logger.With("error", err).Error("Failed to retrieve delete network request by ticket")
		return nil, err
	}

	return &req, nil
}

func GetTicketsWithStatus(status string) ([]Ticket, error) {
	var tickets []Ticket
	err := db.Raw(`SELECT * FROM tickets WHERE uuid IN (SELECT uuid FROM network_requests WHERE status = ? UNION SELECT uuid FROM delete_network_requests WHERE status = ?)`, status, status).Scan(&tickets).Error
	if err != nil {
		logger.With("error", err).Error("Failed to retrieve tickets with status", "status", status)
		return nil, err
	}
	return tickets, nil
}
